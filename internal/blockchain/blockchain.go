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

	DistributeReward(receiver sdk.AccAddress, id uint64, reward uint64) error
}

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

func (b *blockchain) DistributeReward(receiver sdk.AccAddress, id uint64, reward uint64) error {
	msg := pdv.NewMsgDistributeRewards(b.ctx.GetFromAddress(), []pdv.Reward{{
		Receiver: receiver,
		ID:       id,
		Reward:   reward,
	}})
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	return b.BroadcastMsg(msg)
}

func (b *blockchain) BroadcastMsg(msg sdk.Msg) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	txBldr, err := utils.PrepareTxBuilder(b.txBuilder, b.ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare builder: %w", err)
	}

	msgs := []sdk.Msg{msg}

	if txBldr, err = utils.EnrichWithGas(txBldr, b.ctx, msgs); err != nil {
		return errors.New("failed to calculate gas") // nolint: goerr113
	}

	txBytes, err := txBldr.BuildAndSign(b.ctx.GetFromName(), keys.DefaultKeyPass, msgs)
	if err != nil {
		return fmt.Errorf("failed to build and sign tx: %w", err)
	}

	resp, err := b.ctx.BroadcastTx(txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast tx: %w", err)
	}

	if resp.Code != 0 {
		if sdkerrors.ErrTxInMempoolCache.ABCICode() == resp.Code {
			return nil
		}

		if sdkerrors.ErrUnauthorized.ABCICode() == resp.Code || sdkerrors.ErrInvalidSequence.ABCICode() == resp.Code {
			b.txBuilder = b.txBuilder.WithSequence(0) // reset sequence
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
