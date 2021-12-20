// Package blockchain contains code for interacting with the decentr blockchain.
package blockchain

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/Decentr-net/decentr/config"
	operationstypes "github.com/Decentr-net/decentr/x/operations/types"
	"github.com/Decentr-net/go-broadcaster"
)

// nolint: gochecknoinits
func init() {
	config.SetAddressPrefixes()
}

var _ Blockchain = &blockchain{}

//go:generate mockgen -destination=./mock/blockchain.go -package=mock -source=blockchain.go

// Reward is a copy of operations.Reward but with string receiver instead of sdk.AccAddress.
type Reward struct {
	Receiver string
	ID       uint64
	Reward   sdk.Dec
}

// Blockchain is interface for interacting with the blockchain.
type Blockchain interface {
	DistributeRewards(rewards []Reward) (tx string, err error)
}

type blockchain struct {
	b *broadcaster.Broadcaster
}

// New returns new instance of Blockchain.
func New(b *broadcaster.Broadcaster) *blockchain { // nolint:golint
	return &blockchain{
		b: b,
	}
}

func (b blockchain) DistributeRewards(rewards []Reward) (string, error) {
	rr := make([]operationstypes.Reward, len(rewards))

	for i, v := range rewards { // nolint:gocritic
		owner, err := sdk.AccAddressFromBech32(v.Receiver)
		if err != nil {
			return "", fmt.Errorf("invalid receiver: %w", err)
		}

		rr[i] = operationstypes.Reward{
			Receiver: owner.String(),
			Reward:   sdk.DecProto{Dec: v.Reward},
		}
	}

	msg := operationstypes.NewMsgDistributeRewards(b.b.From(), rr)
	if err := msg.ValidateBasic(); err != nil {
		return "", err
	}

	resp, err := b.b.BroadcastMsg(&msg, "")
	if err != nil {
		return "", fmt.Errorf("failed to broadcast MsgDistributeRewards: %w", err)
	}

	return resp.TxHash, nil
}
