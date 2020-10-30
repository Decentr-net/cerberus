// Package s3 contains implementation Storage interface with any s3-compatible storage.
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/cerberus/internal/storage"
)

type s3 struct {
	c *minio.Client
	b string
}

// NewStorage returns s3 implementation of Storage interface.
func NewStorage(client *minio.Client, bucket string) (storage.Storage, error) {
	logrus.WithField("bucket", bucket).Debug("check bucket existence")
	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		logrus.WithField("bucket", bucket).Info("create bucket in s3 storage")
		if err := client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &s3{
		c: client,
		b: bucket,
	}, nil
}

func (s s3) Ping(ctx context.Context) error {
	_, err := s.c.ListBuckets(ctx)
	if err != nil {
		return errors.New("connection with S3 seems broken") // nolint:goerr113
	}
	return nil
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
func (s s3) Write(ctx context.Context, r io.Reader, size int64, path string) error {
	_, err := s.c.PutObject(ctx, s.b, path, r, size, minio.PutObjectOptions{DisableMultipart: true, ContentType: "binary/octet-stream"})
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
