//+build integration

package s3

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Decentr-net/cerberus/internal/storage"
)

const (
	accessKeyID     = "AKIAIOSFODNN7EXAMPLE"
	secretAccessKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
)

var (
	ctx      = context.Background()
	c        *minio.Client
	bucket   = "bucket"
	testFile = "testfile"
)

func TestMain(m *testing.M) {
	shutdown := setup()

	code := m.Run()

	shutdown()
	os.Exit(code)
}

func setup() func() {
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio",
		ExposedPorts: []string{"9000/tcp"},
		WaitingFor:   wait.ForLog("Browser Access:"),
		Env: map[string]string{
			"MINIO_ACCESS_KEY": accessKeyID,
			"MINIO_SECRET_KEY": secretAccessKey,
		},
		Entrypoint: []string{"minio", "server", "/data"},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
	})
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create ipfs node container")
	}

	if err := container.Start(ctx); err != nil {
		logrus.WithError(err).Fatal("failed to start container")
	}

	host, err := container.Host(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("failed to get host")
	}

	port, err := container.MappedPort(ctx, "9000")
	if err != nil {
		logrus.WithError(err).Fatal("failed to map port")
	}

	c, err = minio.New(fmt.Sprintf("%s:%d", host, port.Int()), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		logrus.WithError(err).Fatal("failed to create s3 client")
	}

	if err := c.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		logrus.WithError(err).Fatal("failed to create bucket")
	}

	if _, err := c.PutObject(ctx, bucket, testFile, bytes.NewBufferString("example"), -1, minio.PutObjectOptions{}); err != nil {
		logrus.WithError(err).Fatal("failed to put test file")
	}

	return func() {
		if container == nil {
			return
		}
		if err := container.Terminate(ctx); err != nil {
			logrus.WithError(err).Error("failed to terminate container")
		}
	}
}

func TestS3_Write(t *testing.T) {
	s, err := NewStorage(c, bucket)
	require.NoError(t, err)

	assert.NoError(t, s.Write(ctx, strings.NewReader("example"), "file"))
}

func TestS3_Read(t *testing.T) {
	s, err := NewStorage(c, bucket)
	require.NoError(t, err)

	rc, err := s.Read(ctx, testFile) // text file with "example" word
	require.NoError(t, err)

	b, err := ioutil.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "example", string(b))

	assert.NoError(t, rc.Close())
}

func TestS3_Read_FileNotFound(t *testing.T) {
	s, err := NewStorage(c, bucket)
	require.NoError(t, err)

	rc, err := s.Read(ctx, "not_found")
	assert.Nil(t, rc)
	assert.Error(t, err)
	assert.Equal(t, storage.ErrNotFound, err)
}

func TestS3_Write_Read(t *testing.T) {
	s, err := NewStorage(c, bucket)
	require.NoError(t, err)

	text := []byte("cerberus")

	require.NoError(t, s.Write(ctx, bytes.NewReader(text), "cerberus"))

	rc, err := s.Read(ctx, "cerberus")
	require.NoError(t, err)

	b, err := ioutil.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, text, b)

	assert.NoError(t, rc.Close())
}

func TestS3_DoesExist(t *testing.T) {
	s, err := NewStorage(c, bucket)
	require.NoError(t, err)

	exists, err := s.DoesExist(ctx, testFile) // text file with "example" word
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestS3_DoesExist_NotFound(t *testing.T) {
	s, err := NewStorage(c, bucket)
	require.NoError(t, err)

	exists, err := s.DoesExist(ctx, "not_found")
	assert.Nil(t, err)
	assert.False(t, exists)
}
