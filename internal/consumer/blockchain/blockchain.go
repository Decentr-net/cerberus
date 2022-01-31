// Package blockchain is a consumer interface.
package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/ariadne"
	"github.com/Decentr-net/decentr/config"
	operationstypes "github.com/Decentr-net/decentr/x/operations/types"

	"github.com/Decentr-net/cerberus/internal/consumer"
	"github.com/Decentr-net/cerberus/internal/storage"
)

// nolint:gochecknoinits
func init() {
	config.SetAddressPrefixes()
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
				case *operationstypes.MsgResetAccount:
					err = processMsgResetAccount(ctx, is, b.fs, msg)
				default:
					log.WithField("msg", spew.Sdump(msg)).Debug("skip message")
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

func processMsgResetAccount(ctx context.Context, is storage.IndexStorage, fs storage.FileStorage, msg *operationstypes.MsgResetAccount) error {
	if err := is.DeleteProfile(ctx, msg.Address); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	if err := is.DeletePDV(ctx, msg.Address); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	go func() {
		if err := fs.DeleteData(ctx, msg.Address); err != nil {
			logrus.WithError(err).WithField("account", msg.Address).Error("failed to delete data")
		}
	}()

	return nil
}
