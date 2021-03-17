// Package blockchain contains code for interacting with the decentr blockchain.
package blockchain

import (
	"context"
	"errors"
	"fmt"
	"sync"

	clicontext "github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"

	"github.com/Decentr-net/decentr/app"
	pdv "github.com/Decentr-net/decentr/x/pdv/types"
	logging "github.com/Decentr-net/logrus/context"

	"github.com/Decentr-net/cerberus/internal/health"
)

//go:generate mockgen -destination=./blockchain_mock.go -package=blockchain -source=blockchain.go

// nolint: gochecknoinits
func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	config.Seal()
}

// Blockchain is interface for interacting with the blockchain.
type Blockchain interface {
	health.Pinger

	DistributeReward(ctx context.Context, receiver sdk.AccAddress, id uint64, reward uint64) error
}

var errInvalidSequence = errors.New("invalid sequence_id")

type blockchain struct {
	ctx       clicontext.CLIContext
	txBuilder auth.TxBuilder
	mu        sync.Mutex
}

func NewBlockchain(ctx clicontext.CLIContext, b auth.TxBuilder) Blockchain { // nolint
	return &blockchain{
		ctx:       ctx,
		txBuilder: b,
	}
}

func (b *blockchain) DistributeReward(ctx context.Context, receiver sdk.AccAddress, id uint64, reward uint64) error {
	msg := pdv.NewMsgDistributeRewards(b.ctx.GetFromAddress(), []pdv.Reward{{
		Receiver: receiver,
		ID:       id,
		Reward:   reward,
	}})
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	logging.GetLogger(ctx).WithField("msg", msg).Debug("trying to broadcast msg")
	var err error
	for i := 0; i < 2; i++ {
		err = b.BroadcastMsg(ctx, msg)

		if err == nil {
			return nil
		}

		if errors.Is(err, errInvalidSequence) {
			logging.GetLogger(ctx).WithField("msg", msg).Debug("retry broadcasting msg")
			continue
		}
	}

	return err
}

func (b *blockchain) BroadcastMsg(ctx context.Context, msg sdk.Msg) error {
	log := logging.GetLogger(ctx).WithField("sequence_id", b.txBuilder.Sequence())

	b.mu.Lock()
	defer b.mu.Unlock()

	log.Debug("preparing tx builder")
	txBldr, err := utils.PrepareTxBuilder(b.txBuilder, b.ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare builder: %w", err)
	}

	msgs := []sdk.Msg{msg}

	if txBldr, err = utils.EnrichWithGas(txBldr, b.ctx, msgs); err != nil {
		return errors.New("failed to calculate gas") // nolint: goerr113
	}

	log.Debug("building tx")
	txBytes, err := txBldr.BuildAndSign(b.ctx.GetFromName(), keys.DefaultKeyPass, msgs)
	if err != nil {
		return fmt.Errorf("failed to build and sign tx: %w", err)
	}

	log.Debug("broadcasting tx")
	resp, err := b.ctx.BroadcastTx(txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast tx: %w", err)
	}

	if resp.Code != 0 {
		if sdkerrors.ErrTxInMempoolCache.ABCICode() == resp.Code {
			return nil
		}

		if sdkerrors.ErrUnauthorized.ABCICode() == resp.Code || sdkerrors.ErrInvalidSequence.ABCICode() == resp.Code {
			log.Info("wrong sequence_id. reset sequence_id to zero")
			b.txBuilder = b.txBuilder.WithSequence(0)
			return errInvalidSequence
		}

		return fmt.Errorf("failed to broadcast tx: %s", resp.String()) // nolint: goerr113
	}

	b.txBuilder = b.txBuilder.WithSequence(txBldr.Sequence() + 1)

	return nil
}

func (b *blockchain) Ping(_ context.Context) error {
	c, err := b.ctx.GetNode()
	if err != nil {
		return fmt.Errorf("failed to get rpc client: %w", err)
	}
	if _, err := c.Status(); err != nil {
		return fmt.Errorf("failed to check node status: %w", err)
	}

	return nil
}
