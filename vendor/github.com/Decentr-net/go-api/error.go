package api

import (
	"errors"
	"fmt"
)

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
