// Package broadcaster contains code for interacting with the decentr blockchain.
package broadcaster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	clicontext "github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
)

// ErrTxInMempoolCache is returned when tx is already broadcast and exists in mempool cache.
var ErrTxInMempoolCache = errors.New("tx is already in mempool cache")

var errInvalidSequence = errors.New("invalid sequence")

// Broadcaster provides functionality to broadcast messages to cosmos based blockchain node.
type Broadcaster struct {
	ctx clicontext.CLIContext
	enc sdk.TxEncoder

	genesisKeyPass string
	chainID        string
	num            uint64
	seq            uint64

	mu sync.Mutex
}

// Config ...
type Config struct {
	CLIHome            string
	KeyringBackend     string
	KeyringPromptInput string

	NodeURI       string
	BroadcastMode string

	From           string
	ChainID        string
	GenesisKeyPass string
}

// New returns new instance of broadcaster
func New(cdc *codec.Codec, cfg Config) (*Broadcaster, error) {
	kb, err := keys.NewKeyring(sdk.KeyringServiceName(),
		cfg.KeyringBackend,
		cfg.CLIHome,
		bytes.NewBufferString(cfg.KeyringPromptInput),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	acc, err := kb.Get(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	cliCtx := clicontext.NewCLIContext().
		WithCodec(cdc).
		WithBroadcastMode(cfg.BroadcastMode).
		WithNodeURI(cfg.NodeURI).
		WithFrom(acc.GetName()).
		WithFromName(acc.GetName()).
		WithFromAddress(acc.GetAddress()).
		WithChainID(cfg.ChainID)
	cliCtx.Keybase = kb

	b := &Broadcaster{
		ctx: cliCtx,
		enc: utils.GetTxEncoder(cdc),

		chainID:        cfg.ChainID,
		genesisKeyPass: cfg.GenesisKeyPass,
		mu:             sync.Mutex{},
	}

	if err := b.refreshSequence(); err != nil {
		return nil, fmt.Errorf("failed to refresh sequence: %w", err)
	}

	return b, nil
}

// From returns address of broadcaster.
func (b *Broadcaster) From() sdk.AccAddress {
	return b.ctx.FromAddress
}

// BroadcastMsg broadcasts alone message.
func (b *Broadcaster) BroadcastMsg(msg sdk.Msg, memo string) error {
	return b.Broadcast([]sdk.Msg{msg}, memo)
}

// Broadcast broadcasts messages.
func (b *Broadcaster) Broadcast(msgs []sdk.Msg, memo string) error {
	err := b.broadcast(msgs, memo)

	if errors.Is(err, errInvalidSequence) {
		if err := b.refreshSequence(); err != nil {
			return fmt.Errorf("failed to refresh sequence: %w", err)
		}

		err = b.broadcast(msgs, memo)
	}

	if err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	return nil
}

// PingContext pings node with context.
func (b *Broadcaster) PingContext(ctx context.Context) error {
	err := make(chan error)
	go func() {
		err <- b.Ping()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-err:
		return e
	}
}

// PingContext pings node.
func (b *Broadcaster) Ping() error {
	c, err := b.ctx.GetNode()
	if err != nil {
		return fmt.Errorf("failed to get rpc client: %w", err)
	}
	if _, err := c.ABCIInfo(); err != nil {
		return fmt.Errorf("failed to check node status: %w", err)
	}

	return nil
}

func (b *Broadcaster) broadcast(msgs []sdk.Msg, memo string) error {
	txBldr := auth.NewTxBuilder(b.enc, b.num, b.seq, 0, 1.0, false,
		b.chainID, memo, nil, nil).WithKeybase(b.ctx.Keybase)

	b.mu.Lock()
	defer b.mu.Unlock()

	txBldr, err := utils.PrepareTxBuilder(txBldr, b.ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare builder: %w", err)
	}

	if txBldr, err = utils.EnrichWithGas(txBldr, b.ctx, msgs); err != nil {
		return errors.New("failed to calculate gas")
	}

	txBytes, err := txBldr.BuildAndSign(b.ctx.GetFromName(), b.genesisKeyPass, msgs)
	if err != nil {
		return fmt.Errorf("failed to build and sign tx: %w", err)
	}

	resp, err := b.ctx.BroadcastTx(txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast tx: %w", err)
	}

	if resp.Code != 0 {
		if sdkerrors.ErrTxInMempoolCache.ABCICode() == resp.Code {
			return ErrTxInMempoolCache
		}

		if sdkerrors.ErrUnauthorized.ABCICode() == resp.Code {
			return errInvalidSequence
		}

		return fmt.Errorf("failed to broadcast tx: %s", resp.String())
	}

	b.seq++

	return nil
}

func (b *Broadcaster) refreshSequence() error {
	b.seq, b.num = 0, 0

	var err error

	b.num, b.seq, err = auth.NewAccountRetriever(b.ctx).GetAccountNumberSequence(b.ctx.GetFromAddress())

	return err
}
