// Package blockchain is a consumer interface.
package blockchain

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/ariadne"
	"github.com/Decentr-net/decentr/app"
	"github.com/Decentr-net/decentr/x/operations"

	"github.com/Decentr-net/cerberus/internal/consumer"
	"github.com/Decentr-net/cerberus/internal/storage"
)

// nolint:gochecknoinits
func init() {
	c := sdk.GetConfig()
	c.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	c.Seal()
}

var log = logrus.WithField("package", "blockchain")

type blockchain struct {
	f  ariadne.Fetcher
	fs storage.FileStorage
	is storage.IndexStorage

	retryInterval          time.Duration
	retryLastBlockInterval time.Duration
}

// New returns new blockchain instance.
func New(f ariadne.Fetcher, fs storage.FileStorage, is storage.IndexStorage, retryInterval, retryLastBlockInterval time.Duration) consumer.Consumer {
	return blockchain{
		f:  f,
		fs: fs,
		is: is,

		retryInterval:          retryInterval,
		retryLastBlockInterval: retryLastBlockInterval,
	}
}

func logError(h uint64, err error) {
	log.WithField("height", h).WithError(err).Error("failed to process block")
}

func (b blockchain) Run(ctx context.Context) error {
	from, err := b.is.GetHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}

	return b.f.FetchBlocks(ctx, from, b.processBlockFunc(ctx),
		ariadne.WithErrHandler(logError),
		ariadne.WithSkipError(false),
		ariadne.WithRetryInterval(b.retryInterval),
		ariadne.WithRetryLastBlockInterval(b.retryLastBlockInterval),
	)
}

func (b blockchain) processBlockFunc(ctx context.Context) func(block ariadne.Block) error {
	return func(block ariadne.Block) error {
		return b.is.InTx(ctx, func(is storage.IndexStorage) error {
			log := log.WithField("height", block.Height).WithField("txs", len(block.Txs))
			log.Info("processing block")
			log.WithField("msgs", block.Messages()).Debug()

			for _, msg := range block.Messages() {
				var err error

				switch msg := msg.(type) {
				case operations.MsgResetAccount:
					err = processMsgResetAccount(ctx, is, b.fs, msg)
				default:
					log.WithField("msg", fmt.Sprintf("%s/%s", msg.Route(), msg.Type())).Debug("skip message")
				}

				if err != nil {
					return fmt.Errorf("failed to process msg: %w", err)
				}
			}

			if err := is.SetHeight(ctx, block.Height); err != nil {
				return fmt.Errorf("failed to set height: %w", err)
			}

			return nil
		})
	}
}

func processMsgResetAccount(ctx context.Context, is storage.IndexStorage, fs storage.FileStorage, msg operations.MsgResetAccount) error {
	if err := is.DeleteProfile(ctx, msg.AccountOwner.String()); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	if err := is.DeletePDV(ctx, msg.AccountOwner.String()); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	go func() {
		if err := fs.DeleteData(ctx, msg.AccountOwner.String()); err != nil {
			logrus.WithError(err).WithField("account", msg.AccountOwner).Error("failed to delete data")
		}
	}()

	return nil
}
