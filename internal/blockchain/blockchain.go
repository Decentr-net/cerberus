// Package blockchain contains code for interacting with the decentr blockchain.
package blockchain

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/Decentr-net/decentr/app"
	pdv "github.com/Decentr-net/decentr/x/pdv/types"
	"github.com/Decentr-net/go-broadcaster"
)

// nolint: gochecknoinits
func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	config.Seal()
}

//go:generate mockgen -destination=./mock/blockchain.go -package=mock -source=blockchain.go

// Blockchain is interface for interacting with the blockchain.
type Blockchain interface {
	DistributeReward(receiver sdk.AccAddress, id uint64, reward uint64) error
}

type blockchain struct {
	b *broadcaster.Broadcaster
}

// New returns new instance of Blockchain.
func New(b *broadcaster.Broadcaster) Blockchain {
	return blockchain{
		b: b,
	}
}

func (b blockchain) DistributeReward(receiver sdk.AccAddress, id uint64, reward uint64) error {
	msg := pdv.NewMsgDistributeRewards(b.b.From(), []pdv.Reward{{
		Receiver: receiver,
		ID:       id,
		Reward:   reward,
	}})
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	if err := b.b.BroadcastMsg(msg, ""); err != nil {
		return fmt.Errorf("failed to broadcast MsgDistributeRewards: %w", err)
	}

	return nil
}
