// Package sio contains minio/sio implementation of crypto.Crypto interface.
package sio

import (
	"io"

	"github.com/minio/sio"

	icrypto "github.com/Decentr-net/cerberus/internal/crypto"
)

type crypto struct {
	c sio.Config
}

// NewCrypto returns minio/sio implementation of crypto.Crypto interface.
func NewCrypto(key [32]byte) icrypto.Crypto {
	return &crypto{
		c: sio.Config{
			MinVersion: sio.Version20,
			Key:        key[:],
		},
	}
}

// Encrypt returns reader with encrypted src data.
func (c *crypto) Encrypt(src io.Reader) (io.Reader, error) {
	return sio.EncryptReader(src, c.c)
}

// Encrypt returns reader with decrypted src data.
func (c *crypto) Decrypt(src io.Reader) (io.Reader, error) {
	return sio.DecryptReader(src, c.c)
}
