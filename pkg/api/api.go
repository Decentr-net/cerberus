// Package api provides API and client to Cerberus.
package api

import (
	"context"
	"encoding/gob"
	"errors"
	"time"
)

// SendPDVEndpoint ...
const SendPDVEndpoint = "/v1/send-pdv"

// ReceivePDVEndpoint ...
const ReceivePDVEndpoint = "/v1/receive-pdv"

// DoesPDVExistEndpoint ...
const DoesPDVExistEndpoint = "/v1/pdv-exists"

func init() {
	gob.Register(time.Time{})
}

// ErrInvalidPublicKey is returned when public key is invalid.
var ErrInvalidPublicKey = errors.New("public key is invalid")

// ErrInvalidSignature is returned when signature is invalid.
var ErrInvalidSignature = errors.New("signature is invalid")

// ErrNotFound is returned when object is not found.
var ErrNotFound = errors.New("not found")

// ErrInvalidRequest is returned when request is invalid.
var ErrInvalidRequest = errors.New("invalid request")

// ErrNotVerified is returned when signature is wrong.
var ErrNotVerified = errors.New("failed to verify message")

// Cerberus provides user-friendly API methods.
type Cerberus interface {
	SendPDV(ctx context.Context, data []byte) (string, error)
	ReceivePDV(ctx context.Context, address string) ([]byte, error)
	DoesPDVExist(ctx context.Context, address string) (bool, error)
}
