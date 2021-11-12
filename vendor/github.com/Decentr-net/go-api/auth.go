package api

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

// PublicKeyHeader is name for public key http header.
const PublicKeyHeader = "Public-Key"

// SignatureHeader is name for signature http header.
const SignatureHeader = "Signature"

func GetSignature(r *http.Request) (crypto.PubKey, []byte, error) {
	s, err := hex.DecodeString(r.Header.Get(SignatureHeader))

	if err != nil {
		return nil, nil, ErrInvalidSignature
	}

	k, err := hex.DecodeString(r.Header.Get(PublicKeyHeader))
	if err != nil {
		return nil, nil, ErrInvalidPublicKey
	}

	if len(k) != 33 {
		return nil, nil, ErrInvalidPublicKey
	}

	return secp256k1.PubKey(k), s, nil
}

// Sign signs http request.
func Sign(r *http.Request, pk crypto.PrivKey) error {
	d, err := GetMessageToSign(r)
	if err != nil {
		return fmt.Errorf("failed to get digest: %w", err)
	}

	s, err := pk.Sign(d)
	if err != nil {
		return fmt.Errorf("failed to sign digest: %w", err)
	}

	r.Header.Set(PublicKeyHeader, hex.EncodeToString(pk.PubKey().Bytes())) // truncate amino codec prefix
	r.Header.Set(SignatureHeader, hex.EncodeToString(s))

	return nil
}

// Verify verifies request's signature with public key.
func Verify(r *http.Request) error {
	k, s, err := GetSignature(r)
	if err != nil {
		return err
	}

	d, err := GetMessageToSign(r)
	if err != nil {
		return err
	}
	if !k.VerifySignature(d, s) {
		return ErrNotVerified
	}

	return nil
}

// GetMessageToSign returns message to sign.
func GetMessageToSign(r *http.Request) ([]byte, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	return append(body, []byte(r.URL.Path)...), nil
}

func GetAddressFromPubKey(k string) (sdk.AccAddress, error) {
	b, err := hex.DecodeString(k)
	if err != nil {
		return nil, err
	}

	return sdk.AccAddress(secp256k1.PubKey(b).Address()), err
}
