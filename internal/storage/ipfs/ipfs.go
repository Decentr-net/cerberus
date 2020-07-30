// Package ipfs contains implementation Storage interface with ipfs as storage.
package ipfs

import (
	"context"
	"fmt"
	"io"
	"net/url"

	shell "github.com/ipfs/go-ipfs-api"
	files "github.com/ipfs/go-ipfs-files"

	"github.com/Decentr-net/cerberus/internal/storage"
)

type ipfs struct {
	sh *shell.Shell
}

// NewStorage returns ipfs implementation of Storage interface.
func NewStorage(sh *shell.Shell) storage.Storage {
	return ipfs{
		sh: sh,
	}
}

// Read returns ReadCloser with file content from ipfs node
// It is modified copy of shell.Cat method with use custom context.
func (i ipfs) Read(ctx context.Context, hash string) (io.ReadCloser, error) {
	resp, err := i.sh.Request("cat", hash).Send(ctx)
	if err != nil {
		if err, ok := err.(*url.Error); ok && err.Timeout() && i.sh.IsUp() {
			return nil, storage.ErrNotFound
		}

		return nil, fmt.Errorf("failed to send cat request to ipfs: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("cat request failed: %w", resp.Error)
	}

	return resp.Output, nil
}

// Write puts file into ipfs node
// It is modified copy of shell.Add method with custom context.
func (i ipfs) Write(ctx context.Context, r io.Reader) (string, error) {
	fr := files.NewReaderFile(r)
	slf := files.NewSliceDirectory([]files.DirEntry{files.FileEntry("", fr)})
	fileReader := files.NewMultiFileReader(slf, true)

	rb := i.sh.Request("add")

	var out struct{ Hash string }
	if err := rb.Body(fileReader).Exec(ctx, &out); err != nil {
		return "", fmt.Errorf("failed to add file into ipfs: %w", err)
	}

	return out.Hash, nil
}

// DoesExist checks file's existence in ipfs.
// It is modified copy of shell.ObjectStat method with custom context.
func (i ipfs) DoesExist(ctx context.Context, hash string) (bool, error) {
	resp, err := i.sh.Request("object/stat", hash).Send(ctx)
	if err != nil {
		if err, ok := err.(*url.Error); ok && err.Timeout() && i.sh.IsUp() {
			return false, nil
		}

		return false, fmt.Errorf("failed to send stat request: %w", err)
	}

	if resp.Error != nil {
		return false, fmt.Errorf("stat request failed: %w", resp.Error)
	}

	return true, nil
}
