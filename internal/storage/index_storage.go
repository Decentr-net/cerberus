package storage

import (
	"context"
	"time"
)

//go:generate mockgen -destination=./mock/index_storage.go -package=mock -source=index_storage.go

// IndexStorage provides access to pdv index.
type IndexStorage interface {
	GetProfile(ctx context.Context, addr string) (*Profile, error)
	GetProfiles(ctx context.Context, addr []string) ([]*Profile, error)
	SetProfile(ctx context.Context, p *SetProfileParams) error
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
