//+build integration

package ipfs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Decentr-net/cerberus/internal/storage"
)

var (
	ctx = context.Background()
	sh  *shell.Shell
)

func TestMain(m *testing.M) {
	shutdown := setup()

	code := m.Run()

	shutdown()
	os.Exit(code)
}

func setup() func() {
	req := testcontainers.ContainerRequest{
		Image:        "ipfs/go-ipfs:latest",
		ExposedPorts: []string{"5001/tcp", "4001/tcp"},
		WaitingFor:   wait.ForListeningPort("5001/tcp"),
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

	port, err := container.MappedPort(ctx, "5001")
	if err != nil {
		logrus.WithError(err).Fatal("failed to map port")
	}

	sh = shell.NewShellWithClient(fmt.Sprintf("%s:%d", host, port.Int()), &http.Client{Timeout: time.Second})

	if _, err := sh.StatsBW(ctx); err != nil {
		logrus.WithError(err).Fatal("failed to get bandwidth stats")
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

func TestIpfs_Write(t *testing.T) {
	i := NewStorage(sh)

	hash, err := i.Write(ctx, strings.NewReader("example"))

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestIpfs_Write_UnavailableNode(t *testing.T) {
	i := NewStorage(shell.NewShell(""))

	hash, err := i.Write(ctx, strings.NewReader("example"))

	assert.Error(t, err)
	assert.Empty(t, hash)
}

func TestIpfs_Read(t *testing.T) {
	i := NewStorage(sh)

	rc, err := i.Read(ctx, "QmcrBuzrkvb4LPPcgFeH4NSvCWvABPBQtSod2PNHQnLJgV") // text file with "example" word
	require.NoError(t, err)

	b, err := ioutil.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "example", string(b))

	assert.NoError(t, rc.Close())
}

func TestIpfs_Read_UnavailableNode(t *testing.T) {
	i := NewStorage(shell.NewShell(""))

	rc, err := i.Read(ctx, "QmcrBuzrkvb4LPPcgFeH4NSvCWvABPBQtSod2PNHQnLJgV")
	assert.Nil(t, rc)
	assert.Error(t, err)
}

func TestIpfs_Read_FileNotFound(t *testing.T) {
	i := NewStorage(sh)

	rc, err := i.Read(ctx, "QmcrBuzrkvb4LPPcgFeH4NSvCWvABPBQtSod2PNHQnLJgN")
	assert.Nil(t, rc)
	assert.Error(t, err)
	assert.Equal(t, storage.ErrNotFound, err)
}

func TestIpfs_Write_Read(t *testing.T) {
	i := NewStorage(sh)

	text := []byte("cerberus")
	hash, err := i.Write(ctx, bytes.NewReader(text))

	require.NoError(t, err)
	require.NotEmpty(t, hash)

	rc, err := i.Read(ctx, hash)
	require.NoError(t, err)

	b, err := ioutil.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, text, b)

	assert.NoError(t, rc.Close())
}

func TestIpfs_DoesExist(t *testing.T) {
	i := NewStorage(sh)

	exists, err := i.DoesExist(ctx, "QmcrBuzrkvb4LPPcgFeH4NSvCWvABPBQtSod2PNHQnLJgV") // text file with "example" word
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestIpfs_DoesExist_NotFound(t *testing.T) {
	i := NewStorage(sh)

	exists, err := i.DoesExist(ctx, "QmcrBuzrkvb4LPPcgFeH4NSvCWvABPBQtSod2PNHQnLJgN")
	assert.Nil(t, err)
	assert.False(t, exists)
}

func TestIpfs_DoesExist_UnavailableNode(t *testing.T) {
	i := NewStorage(shell.NewShell(""))

	exists, err := i.DoesExist(ctx, "QmcrBuzrkvb4LPPcgFeH4NSvCWvABPBQtSod2PNHQnLJgN")
	assert.False(t, exists)
	assert.Error(t, err)
}
