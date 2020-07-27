// Package auth provides authentication mechanism.
package auth

import (
	"encoding/base64"
	"errors"
)

// ErrInvalidPublicKey is returned when public key is invalid.
var ErrInvalidPublicKey = errors.New("public key is invalid")

// ErrInvalidSignature is returned when signature is invalid.
var ErrInvalidSignature = errors.New("signature is invalid")

// Request interface contains all necessary methods for Verify func.
type Request interface {
	// GetPublicKey returns public key.
	GetPublicKey() ([]byte, error)
	// GetSignature returns signature.
	GetSignature() ([]byte, error)
	// GetDigest returns request digest.
	GetDigest() []byte
}

// Signature contains signature and public key for proving it.
type Signature struct {
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

// BareRequest is partial implementation of Request interface. You have to embed this struct into request
// and implement only GetDigest method.
type BareRequest struct {
	Signature Signature `json:"Signature"`
}

// GetPublicKey returns public key.
func (r *BareRequest) GetPublicKey() ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(r.Signature.PublicKey)

	if err != nil {
		return nil, ErrInvalidPublicKey
	}

	return b, nil
}

// GetSignature returns signature.
func (r *BareRequest) GetSignature() ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(r.Signature.Signature)

	if err != nil {
		return nil, ErrInvalidSignature
	}

	return b, nil
}
