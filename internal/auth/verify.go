package auth

import (
	"errors"

	amino "github.com/tendermint/tendermint/crypto/encoding/amino"
)

// ErrUnauthenticated is returned when signature is wrong.
var ErrUnauthenticated = errors.New("failed to verify message")

// Verify verifies message's signature with public key.
func Verify(r Request) error {
	b, err := r.GetPublicKey()
	if err != nil {
		return err
	}

	k, err := amino.PubKeyFromBytes(b)
	if err != nil {
		return ErrInvalidPublicKey
	}

	b, err = r.GetSignature()
	if err != nil {
		return err
	}
	if !k.VerifyBytes(r.GetDigest(), b) {
		return ErrUnauthenticated
	}

	return nil
}
