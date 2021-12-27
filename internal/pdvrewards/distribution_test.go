package pdvrewards

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	blockchainmock "github.com/Decentr-net/cerberus/internal/blockchain/mock"
	"github.com/Decentr-net/cerberus/internal/storage"
	"github.com/Decentr-net/cerberus/internal/storage/mock"
)

func TestDistributor_prepareRewardsQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	delta1 := storage.PDVDelta{
		Address: "addr1",
		Delta:   30,
	}

	delta2 := storage.PDVDelta{
		Address: "addr2",
		Delta:   20,
	}

	b := blockchainmock.NewMockBlockchain(ctrl)
	is := mock.NewMockIndexStorage(ctrl)
	is.EXPECT().GetPDVDeltaList(gomock.Any()).Return([]*storage.PDVDelta{&delta1, &delta2}, nil)
	is.EXPECT().InTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f func(_ storage.IndexStorage) error) error {
		return f(is)
	})

	// asset 2 reward queue items created
	is.EXPECT().CreateRewardsQueueItem(gomock.Any(), delta1.Address, int64(600))
	is.EXPECT().CreateRewardsQueueItem(gomock.Any(), delta2.Address, int64(400))

	// asset distributed date is set
	is.EXPECT().SetPDVRewardsDistributedDate(gomock.Any(), gomock.Any()).Do(func(_ context.Context, date time.Time) {
		require.Equal(t, time.UTC, date.Location())
	})

	d := NewDistributor(1000, b, is)

	//act
	d.prepareRewardsQueue()
}

func TestDistributor_distributeRewardsIfExist(t *testing.T) {
	const itemsCount = 100

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	is := mock.NewMockIndexStorage(ctrl)

	items := make([]*storage.RewardsQueueItem, itemsCount)
	for i := 0; i < 100; i++ {
		items[i] = &storage.RewardsQueueItem{
			Address: fmt.Sprint("address", i+1),
			Reward:  int64(i + 1),
		}
	}
	is.EXPECT().GetRewardsQueueItemList(gomock.Any()).Return(items, nil)
	is.EXPECT().InTx(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, f func(_ storage.IndexStorage) error) error {
		return f(is)
	}).Times(itemsCount / chunkSize)

	for _, item := range items {
		is.EXPECT().DeleteRewardsQueueItem(gomock.Any(), item.Address)
	}

	b := blockchainmock.NewMockBlockchain(ctrl)
	b.EXPECT().SendStakes(gomock.Any(), rewardsMemo).Return(nil).Times(itemsCount / chunkSize)

	d := NewDistributor(1000, b, is)

	//act
	d.distributeRewardsIfExist()
}
