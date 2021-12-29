// Package pdvrewards contains code for rewards calculation and distribution
package pdvrewards

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/storage"
)

const (
	chunkSize   = 25
	rewardsMemo = "PDV rewards"
)

var log = logrus.WithField("package", "pdvrewards")

type reward struct {
	address string
	reward  int64
}

// Distributor responsible for distributing PDV rewards in uDEC.
type Distributor struct {
	rewardsPoolSize int64

	b  blockchain.Blockchain
	is storage.IndexStorage
}

// NewDistributor creates a new instance of Distributor.
func NewDistributor(
	rewardsPoolSize int64,
	b blockchain.Blockchain,
	is storage.IndexStorage) *Distributor {
	return &Distributor{
		rewardsPoolSize: rewardsPoolSize,
		b:               b,
		is:              is,
	}
}

// prepareRewardsQueue prepares rewards queue in storage.
func (d *Distributor) prepareRewardsQueue() {
	ctx := context.Background()

	deltas, err := d.is.GetPDVDeltaList(ctx)
	if err != nil {
		log.WithError(err).Error("failed to get PDV delta list")
		return
	}

	var total float64
	for _, delta := range deltas {
		total += delta.Delta
	}

	if total == 0 {
		log.Info("total PDV delta is zero")
		return
	}

	var rewards []reward
	for _, delta := range deltas {
		r := int64(delta.Delta * float64(d.rewardsPoolSize) / total)

		if r != 0 {
			rewards = append(rewards, reward{
				address: delta.Address,
				reward:  r,
			})
		}
	}

	if err := d.is.InTx(ctx, func(tx storage.IndexStorage) error {
		for _, r := range rewards {
			if err1 := tx.CreateRewardsQueueItem(ctx, r.address, r.reward); err1 != nil {
				return fmt.Errorf("failed to create rewards quque item: %w", err1)
			}
		}

		return tx.SetPDVRewardsDistributedDate(ctx, time.Now().UTC())
	}); err != nil {
		log.WithError(err).Error("failed to prepare rewards")
	}
}

// distributeRewardsIfExist distributes rewards if any item exists in queue.
func (d *Distributor) distributeRewardsIfExist() {
	ctx := context.Background()

	items, err := d.is.GetRewardsQueueItemList(ctx)
	if err != nil {
		log.WithError(err).Error("failed to get rewards queue item list")
		return
	}

	if len(items) == 0 {
		log.Debug("queue is empty")
		return
	}

	log.Infof("%d rewards to distribute", len(items))

	chunks := chunkSlice(items, chunkSize)
	for _, chunk := range chunks {
		if err := d.sendStakes(chunk); err != nil {
			log.WithError(err).Error("failed to send stakes")
			return
		}
		if err := d.deleteItemsFromQueue(ctx, chunk); err != nil {
			log.WithError(err).Error("failed to delete item form queue`")
			return
		}

		for _, item := range chunk {
			log.Infof("%s got %d uDEC", item.Address, item.Reward)
		}
	}

	log.Infof("%d rewards distributed", len(items))
}

func (d *Distributor) sendStakes(items []*storage.RewardsQueueItem) error {
	stakes := make([]blockchain.Stake, len(items))
	for idx, item := range items {
		stakes[idx] = blockchain.Stake{
			Address: item.Address,
			Amount:  sdk.NewInt(item.Reward),
		}
	}
	return d.b.SendStakes(stakes, rewardsMemo)
}

func (d *Distributor) deleteItemsFromQueue(ctx context.Context, items []*storage.RewardsQueueItem) error {
	return d.is.InTx(ctx, func(tx storage.IndexStorage) error {
		for _, item := range items {
			if err := tx.DeleteRewardsQueueItem(ctx, item.Address); err != nil {
				return err
			}
		}
		return nil
	})
}

func chunkSlice(slice []*storage.RewardsQueueItem, chunkSize int) [][]*storage.RewardsQueueItem {
	var chunks [][]*storage.RewardsQueueItem
	for {
		if len(slice) == 0 {
			break
		}

		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}

		chunks = append(chunks, slice[0:chunkSize])
		slice = slice[chunkSize:]
	}

	return chunks
}

// RunAsync times to check if distribute date is reached and whether to distribute rewards.
func (d *Distributor) RunAsync(ctx context.Context, interval time.Duration) {
	// if date exceeded prepare a distribution queue
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			date, err := d.is.GetPDVRewardsDistributedDate(ctx)
			if err != nil {
				log.WithError(err).Error("failed to get pdv rewards distributed date")
				return
			}

			if time.Now().After(date.Add(interval)) {
				// PDVRewardsDistributedDate is updated in transaction
				d.prepareRewardsQueue()
			}
		}
	}()

	// if any item in distribution queue, send a reward
	go func() {
		d.distributeRewardsIfExist()

		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			d.distributeRewardsIfExist()
		}
	}()
}
