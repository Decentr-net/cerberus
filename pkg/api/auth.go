package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"reflect"
	"unicode"

	amino "github.com/tendermint/tendermint/crypto/encoding/amino"
)

// Signature contains signature and public key for proving it.
type Signature struct {
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

// AuthRequest is signature and public key keeper.
type AuthRequest struct {
	Signature Signature `json:"signature"`
}

// signatureGetter used in Verify method.
type signatureGetter interface {
	// GetPublicKey returns public key.
	GetPublicKey() ([]byte, error)
	// GetSignature returns signature.
	GetSignature() ([]byte, error)
}

// signatureSetter used in signRequest to set signature in request.
type signatureSetter interface {
	setSignature(s Signature)
}

func (r *AuthRequest) setSignature(s Signature) {
	r.Signature = s
}

// GetPublicKey returns public key.
func (r *AuthRequest) GetPublicKey() ([]byte, error) {
	b, err := hex.DecodeString(r.Signature.PublicKey)

	if err != nil {
		return nil, ErrInvalidPublicKey
	}

	return b, nil
}

// GetSignature returns signature.
func (r *AuthRequest) GetSignature() ([]byte, error) {
	b, err := hex.DecodeString(r.Signature.Signature)

	if err != nil {
		return nil, ErrInvalidSignature
	}

	return b, nil
}

// Verify verifies request's signature with public key.
func Verify(r signatureGetter) error {
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

	d, err := Digest(r)
	if err != nil {
		return err
	}
	if !k.VerifyBytes(d, b) {
		return ErrNotVerified
	}

	return nil
}

// Digest returns sha256 of request.
// Digest will not be taken from unexported and non-marshaling(json) fields.
func Digest(r interface{}) ([]byte, error) {
	v := reflect.ValueOf(r)

	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	b := bytes.NewBuffer([]byte{})
	e := gob.NewEncoder(b)

	if v.Type().Kind() != reflect.Struct {
		panic("digest only can be taken from struct")
	}

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		t := v.Type().Field(i)

		// We want to skip embedded AuthRequest and fields which won't be marshaled by JsonMarshaller.
		if f.Type().Name() == reflect.TypeOf(AuthRequest{}).Name() ||
			isUnexportedField(&t) ||
			t.Tag.Get("json") == "-" {
			continue
		}

		if err := e.EncodeValue(f); err != nil {
			return nil, fmt.Errorf("failed to get digest from %s: %w", v.Type().Field(i).Name, err)
		}
	}

	d := sha256.Sum256(b.Bytes())

	return d[:], nil
}

func isUnexportedField(t *reflect.StructField) bool {
	return unicode.IsLower(rune(t.Name[0]))
}
