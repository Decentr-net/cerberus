// Package storage contains Storage interface and its mock.
package storage

import (
	"context"
	"io"
)

//go:generate mockgen -destination=./storage_mock.go -package=storage -source=storage.go

// Storage is interface which provides access to user's data blocks.
type Storage interface {
	Read(ctx context.Context, hash string) (io.ReadCloser, error)
	Write(ctx context.Context, r io.Reader) (string, error)
}
