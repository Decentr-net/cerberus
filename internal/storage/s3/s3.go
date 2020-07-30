// Package s3 contains implementation Storage interface with any s3-compatible storage.
package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"

	"github.com/Decentr-net/cerberus/internal/storage"
)

type s3 struct {
	c *minio.Client
	b string
}

// NewStorage returns s3 implementation of Storage interface.
func NewStorage(client *minio.Client, bucket string) storage.Storage {
	return &s3{
		c: client,
		b: bucket,
	}
}

// Read returns ReadCloser with file content from s3 storage.
func (s s3) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	r, err := s.c.GetObject(ctx, s.b, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	if _, err := r.Stat(); err != nil {
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get reader stats: %w", err)
	}

	return r, nil
}

// Write puts file into s3 storage.
func (s s3) Write(ctx context.Context, r io.Reader, path string) error {
	_, err := s.c.PutObject(ctx, s.b, path, r, -1, minio.PutObjectOptions{ContentType: "binary/octet-stream"})

	return err
}

// DoesExist checks file's existence in s3 storage.
func (s s3) DoesExist(ctx context.Context, path string) (bool, error) {
	_, err := s.c.StatObject(ctx, s.b, path, minio.StatObjectOptions{})

	if err != nil {
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("get object failed: %w", err)
	}

	return true, nil
}
