// Package ariadne is a library for fetching blocks from cosmos based blockchain node.
package ariadne

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/spm/cosmoscmd"
	tt "github.com/tendermint/tendermint/proto/tendermint/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/Decentr-net/decentr/app"
)

// ErrTooHighBlockRequested returned when blockchain's height is less than requested.
var ErrTooHighBlockRequested = errors.New("too high block requested")

// Block presents transactions and height.
// If you need have more information open new issue on github or DIY and send pull request.
type Block struct {
	Height uint64
	Time   time.Time
	Txs    []sdk.Tx
}

//go:generate mockgen -destination=./mock/ariadne_mock.go -package=mock -source=ariadne.go

// Fetcher interface for fetching.
type Fetcher interface {
	// FetchBlocks starts fetching routine and runs handleFunc for every block.
	FetchBlocks(ctx context.Context, from uint64, handleFunc func(b Block) error, opts ...FetchBlocksOption) error
	// FetchBlock fetches block from blockchain.
	// If height is zero then the highest block will be requested.
	FetchBlock(ctx context.Context, height uint64) (*Block, error)
}

type fetcher struct {
	c       tmservice.ServiceClient
	d       sdk.TxDecoder
	timeout time.Duration
}

// New returns new instance of fetcher.
func New(ctx context.Context, node string, timeout time.Duration) (Fetcher, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, node, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc conn: %w", err)
	}

	return fetcher{
		c:       tmservice.NewServiceClient(conn),
		d:       cosmoscmd.MakeEncodingConfig(app.ModuleBasics).TxConfig.TxDecoder(),
		timeout: timeout,
	}, nil
}

// FetchBlocks starts fetching routine and runs handleFunc for every block.
func (f fetcher) FetchBlocks(ctx context.Context, from uint64, handleFunc func(b Block) error, opts ...FetchBlocksOption) error {
	cfg := defaultFetchBlockOptions
	for _, v := range opts {
		v(&cfg)
	}

	height := uint64(1)
	if from > 0 {
		height = from
	}

	var (
		b   *Block
		err error
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if b == nil {
				if b, err = f.FetchBlock(ctx, height); err != nil {
					if errors.Is(err, ErrTooHighBlockRequested) {
						time.Sleep(cfg.retryLastBlockInterval)
						continue
					}

					cfg.errHandler(height, fmt.Errorf("failed to get block: %w", err))
					time.Sleep(cfg.retryInterval)
					continue
				}
			}

			if err := handleFunc(*b); err != nil {
				cfg.errHandler(b.Height, err)
				if !cfg.skipError {
					time.Sleep(cfg.retryInterval)
					continue
				}
			}

			b = nil
			height++
		}
	}
}

// FetchBlock fetches block from blockchain.
// If height is zero then the highest block will be requested.
func (f fetcher) FetchBlock(ctx context.Context, height uint64) (*Block, error) {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	var block *tt.Block
	if height == 0 {
		res, err := f.c.GetLatestBlock(ctx, &tmservice.GetLatestBlockRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block: %w", err)
		}
		block = res.Block
	} else {
		res, err := f.c.GetBlockByHeight(ctx, &tmservice.GetBlockByHeightRequest{Height: int64(height)})
		if err != nil {
			if err, ok := status.FromError(err); ok {
				if strings.Contains(err.Message(), "requested block height is bigger then the chain length") {
					return nil, ErrTooHighBlockRequested
				}
			}
			return nil, fmt.Errorf("failed to get block: %w", err)
		}
		block = res.Block
	}

	txs := make([]sdk.Tx, len(block.Data.Txs))
	for i, v := range block.Data.Txs {
		tx, err := f.d(v)
		if err != nil {
			return nil, fmt.Errorf("failed to decode tx: %w", err)
		}
		txs[i] = tx
	}

	return &Block{
		Height: uint64(block.Header.Height),
		Time:   block.Header.Time,
		Txs:    txs,
	}, nil
}

// Messages returns all messages in all transactions.
func (b Block) Messages() []sdk.Msg {
	msgs := make([]sdk.Msg, 0, len(b.Txs))
	for _, tx := range b.Txs {
		msgs = append(msgs, tx.GetMsgs()...)
	}

	return msgs
}
