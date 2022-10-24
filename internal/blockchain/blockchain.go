// Package blockchain contains code for interacting with the decentr blockchain.
package blockchain

import (
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/Decentr-net/decentr/config"
	operationstypes "github.com/Decentr-net/decentr/x/operations/types"
	"github.com/Decentr-net/go-broadcaster"
)

// nolint: gochecknoinits
func init() {
	config.SetAddressPrefixes()
}

var _ Blockchain = &blockchain{}

var log = logrus.WithField("package", "blockchain")

//go:generate mockgen -destination=./mock/blockchain.go -package=mock -source=blockchain.go

// ErrInvalidAddress is returned when address is invalid. It is unexpected situation.
var ErrInvalidAddress = errors.New("invalid address")

// Reward is a copy of operations.Reward but with string receiver instead of sdk.AccAddress.
type Reward struct {
	Receiver string
	ID       uint64
	Reward   sdk.Dec
}

// Stake ...
type Stake struct {
	Address string
	Amount  sdk.Int
}

// Blockchain is interface for interacting with the blockchain.
type Blockchain interface {
	DistributeRewards(rewards []Reward) (tx string, err error)
	SendStakes(stakes []Stake, memo string) error
}

type blockchain struct {
	b broadcaster.Broadcaster
}

// New returns new instance of Blockchain.
func New(b broadcaster.Broadcaster) *blockchain { // nolint:golint
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

	log := log.WithFields(logrus.Fields{
		"rewards": rr,
	})

	log.Info("DistributeRewards")

	var resp *sdk.TxResponse
	if err := retry.Do(func() error {
		var err1 error
		resp, err1 = b.b.BroadcastMsg(&msg, "")
		if err1 != nil {
			log.WithError(err1).Warn("attempt to  broadcast MsgDistributeRewards failed")
		}
		return err1
	}, retry.Attempts(3), retry.Delay(200*time.Millisecond)); err != nil {
		return "", fmt.Errorf("failed to broadcast MsgDistributeRewards: %w", err)
	}

	return resp.TxHash, nil
}

// SendStakes ...
func (b blockchain) SendStakes(stakes []Stake, memo string) error {
	sendStakes := func() error {
		messages := make([]sdk.Msg, len(stakes))
		for idx, stake := range stakes {
			to, err := sdk.AccAddressFromBech32(stake.Address)
			if err != nil {
				return fmt.Errorf("%w: %s", ErrInvalidAddress, stake.Address)
			}

			messages[idx] = banktypes.NewMsgSend(b.b.From(), to, sdk.Coins{sdk.Coin{
				Denom:  config.DefaultBondDenom,
				Amount: stake.Amount,
			}})
			if err := messages[idx].ValidateBasic(); err != nil {
				return err
			}
		}

		if _, err := b.b.Broadcast(messages, memo); err != nil {
			return fmt.Errorf("failed to broadcast msg: %w", err)
		}

		return nil
	}

	return retry.Do(sendStakes, retry.Attempts(5), retry.Delay(time.Second))
}
