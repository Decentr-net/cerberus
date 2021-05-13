package storage

import (
	"context"
	"time"
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
	DeleteProfile(ctx context.Context, addr string) error
}

// Profile ...
type Profile struct {
	Address   string
	FirstName string
	LastName  string
	Bio       string
	Avatar    string
	Gender    string
	Birthday  time.Time
	UpdatedAt *time.Time
	CreatedAt time.Time
}

// SetProfileParams ...
type SetProfileParams struct {
	Address   string
	FirstName string
	LastName  string
	Bio       string
	Avatar    string
	Gender    string
	Birthday  time.Time
}
