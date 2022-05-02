package storage

import (
	"context"
	"time"

	"github.com/Decentr-net/cerberus/internal/entities"
)

//go:generate mockgen -destination=./mock/index_storage.go -package=mock -source=index_storage.go

// IndexStorage provides access to pdv index.
type IndexStorage interface {
	InTx(ctx context.Context, f func(s IndexStorage) error) error
	SetHeight(ctx context.Context, height uint64) error
	GetHeight(ctx context.Context) (uint64, error)

	GetProfile(ctx context.Context, addr string) (*Profile, error)
	GetProfiles(ctx context.Context, addr []string) ([]*Profile, error)
	SetProfile(ctx context.Context, p *SetProfileParams) error
	SetProfileBanned(ctx context.Context, addr string) error
	IsProfileBanned(ctx context.Context, addr string) (bool, error)
	DeleteProfile(ctx context.Context, addr string) error

	ListPDV(ctx context.Context, owner string, from uint64, limit uint16) ([]uint64, error)
	DeletePDV(ctx context.Context, owner string) error

	GetPDVMeta(ctx context.Context, address string, id uint64) (*entities.PDVMeta, error)
	SetPDVMeta(ctx context.Context, address string, id uint64, tx string, device string, m *entities.PDVMeta) error

	GetPDVDelta(ctx context.Context, address string) (float64, error)
	GetPDVTotalDelta(ctx context.Context) (float64, error)
	GetPDVDeltaList(ctx context.Context) ([]*PDVDelta, error)

	CreateRewardsQueueItem(ctx context.Context, addr string, reward int64) error
	GetRewardsQueueItemList(ctx context.Context) ([]*RewardsQueueItem, error)
	DeleteRewardsQueueItem(ctx context.Context, addr string) error

	GetPDVRewardsDistributedDate(ctx context.Context) (time.Time, error)
	SetPDVRewardsDistributedDate(ctx context.Context, date time.Time) error
}

// PDVDelta ...
type PDVDelta struct {
	Address string
	Delta   float64
}

// RewardsQueueItem ...
type RewardsQueueItem struct {
	Address string
	Reward  int64
}

// Profile ...
type Profile struct {
	Address   string
	FirstName string
	LastName  string
	Emails    []string
	Bio       string
	Avatar    string
	Gender    string
	Banned    bool
	Birthday  *time.Time
	UpdatedAt *time.Time
	CreatedAt time.Time
}

// SetProfileParams ...
type SetProfileParams struct {
	Address   string
	FirstName string
	LastName  string
	Emails    []string
	Bio       string
	Avatar    string
	Gender    string
	Birthday  *time.Time
}
