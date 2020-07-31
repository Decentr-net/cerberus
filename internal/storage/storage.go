// Package storage contains Storage interface and its mock.
package storage

import (
	"context"
	"errors"
	"io"
)

//go:generate mockgen -destination=./storage_mock.go -package=storage -source=storage.go

// ErrNotFound means that file is not found.
var ErrNotFound = errors.New("not found")

// Storage is interface which provides access to user's data.
type Storage interface {
	Read(ctx context.Context, path string) (io.ReadCloser, error)
	Write(ctx context.Context, r io.Reader, path string) error
	DoesExist(ctx context.Context, path string) (bool, error)
}
