// Package api provides API and client to Cerberus.
package api

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"time"
)

//go:generate mockgen -destination=./api_mock.go -package=api -source=api.go

// SendPDVEndpoint ...
const SendPDVEndpoint = "/v1/send-pdv"

// ReceivePDVEndpoint ...
const ReceivePDVEndpoint = "/v1/receive-pdv"

// DoesPDVExistEndpoint ...
const DoesPDVExistEndpoint = "/v1/pdv-exists"

func init() {
	gob.Register(time.Time{})
}

// ErrInvalidRequest is returned when request is invalid.
var ErrInvalidRequest = errors.New("invalid request")

// ErrInvalidPublicKey is returned when public key is invalid.
var ErrInvalidPublicKey = fmt.Errorf("%w: public key is invalid", ErrInvalidRequest)

// ErrInvalidSignature is returned when signature is invalid.
var ErrInvalidSignature = fmt.Errorf("%w: signature is invalid", ErrInvalidRequest)

// ErrNotFound is returned when object is not found.
var ErrNotFound = errors.New("not found")

// ErrNotVerified is returned when signature is wrong.
var ErrNotVerified = errors.New("failed to verify message")

// Cerberus provides user-friendly API methods.
type Cerberus interface {
	SendPDV(ctx context.Context, data []byte) (string, error)
	ReceivePDV(ctx context.Context, address string) ([]byte, error)
	DoesPDVExist(ctx context.Context, address string) (bool, error)
}
