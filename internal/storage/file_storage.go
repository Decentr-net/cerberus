// Package storage contains FileStorage interface and its mock.
package storage

import (
	"context"
	"errors"
	"io"

	"github.com/Decentr-net/cerberus/internal/health"
)

//go:generate mockgen -destination=./mock/file_storage.go -package=mock -source=file_storage.go

// ErrNotFound means that file is not found.
var ErrNotFound = errors.New("not found")

// FileStorage is interface which provides access to user's data.
type FileStorage interface {
	health.Pinger

	List(ctx context.Context, prefix string, from uint64, limit uint16) ([]string, error)
	Read(ctx context.Context, path string) (io.ReadCloser, error)
	Write(ctx context.Context, data io.Reader, size int64, path string, contentType string, isPublicRead bool) (string, error)

	DeleteData(ctx context.Context, address string) error
}
