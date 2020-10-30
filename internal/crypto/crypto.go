// Package crypto contains encrypting and decrypting reader.
package crypto

import "io"

//go:generate mockgen -destination=./crypto_mock.go -package=crypto -source=crypto.go

// Crypto provide Reader and Writer wrappers.
type Crypto interface {
	// Encrypt returns reader with encrypted src data and size of encrypted data.
	Encrypt(io.Reader) (io.Reader, int64, error)
	// Decrypt returns reader with decrypted src data.
	Decrypt(io.Reader) (io.Reader, error)
}
