// Package s3 contains implementation FileStorage interface with any s3-compatible storage.
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"

	"github.com/Decentr-net/cerberus/internal/storage"
)

type s3 struct {
	c *minio.Client
	b string
}

// NewStorage returns s3 implementation of FileStorage interface.
func NewStorage(client *minio.Client, bucket string) (storage.FileStorage, error) {
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
func (s s3) Write(ctx context.Context, r io.Reader, size int64, path string, contentType string) (string, error) {
	i, err := s.c.PutObject(ctx, s.b, path, r, size, minio.PutObjectOptions{DisableMultipart: true, ContentType: contentType})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", i.Bucket, i.Key), nil
}

// List returns objects by prefix with paging.
func (s s3) List(ctx context.Context, prefix string, from uint64, limit uint16) ([]string, error) {
	to := from + uint64(limit)

	ch := s.c.ListObjects(ctx, s.b, minio.ListObjectsOptions{
		Prefix:    fmt.Sprintf("%s/", prefix),
		Recursive: true,
	})

	i := uint64(0)

	out := make([]string, 0)
	for v := range ch {
		i++
		if i > to {
			return out, nil
		}
		if i <= from {
			continue
		}
		out = append(out, v.Key[len(prefix)+1:])
	}

	return out, nil
}

// DeleteData ...
func (s s3) DeleteData(ctx context.Context, address string) error {
	ch := s.c.ListObjects(ctx, s.b, minio.ListObjectsOptions{
		Prefix:    fmt.Sprintf("%s/", address),
		Recursive: true,
	})

	b := strings.Builder{}
	for err := range s.c.RemoveObjects(ctx, s.b, ch, minio.RemoveObjectsOptions{}) {
		b.WriteString(fmt.Sprintf("failed to remove %s: %s\n", err.ObjectName, err.Err.Error()))
	}

	if b.String() != "" {
		return errors.New(b.String())
	}

	return nil
}
