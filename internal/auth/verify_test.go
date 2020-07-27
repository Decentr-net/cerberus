package auth

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSignature = "SZIO1hiOb0iLJberEAAkU0JVvzOAr/D9ZUf+1MbUhexlNwn14iODT2Sqr2bj3oeHYd9Zqo4aJiQaDk+KaMq+UA=="

type TestAuthRequest struct {
	BareRequest
	FilePath string
}

func (r *TestAuthRequest) GetDigest() []byte {
	return []byte(r.FilePath)
}

func TestVerify(t *testing.T) {
	filepath := "somefile"

	key := secp256k1.GenPrivKey()
	signature, err := key.Sign([]byte(filepath))
	require.NoError(t, err)

	pub := key.PubKey()

	r := TestAuthRequest{
		FilePath: filepath,
		BareRequest: BareRequest{
			Signature: Signature{
				PublicKey: base64.StdEncoding.EncodeToString(pub.Bytes()),
				Signature: base64.StdEncoding.EncodeToString(signature),
			},
		},
	}

	require.NoError(t, Verify(&r))
}

func TestVerify_InvalidKey(t *testing.T) {
	r := TestAuthRequest{
		BareRequest: BareRequest{
			Signature: Signature{
				PublicKey: "wrong key",
				Signature: testSignature,
			},
		},
	}

	err := Verify(&r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPublicKey))
}

func TestVerify_InvalidKey2(t *testing.T) {
	r := TestAuthRequest{
		BareRequest: BareRequest{
			Signature: Signature{
				PublicKey: testSignature,
				Signature: testSignature,
			},
		},
	}

	err := Verify(&r)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPublicKey))
}

func TestVerify_InvalidSignature(t *testing.T) {
	r := TestAuthRequest{
		BareRequest: BareRequest{
			Signature: Signature{
				PublicKey: base64.StdEncoding.EncodeToString(secp256k1.GenPrivKey().PubKey().Bytes()),
				Signature: "not base64",
			},
		},
	}

	err := Verify(&r)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidSignature))
}

func TestVerify_WrongSignature(t *testing.T) {
	r := TestAuthRequest{
		FilePath: "myfile",
		BareRequest: BareRequest{
			Signature: Signature{
				PublicKey: base64.StdEncoding.EncodeToString(secp256k1.GenPrivKey().PubKey().Bytes()),
				Signature: "SZIO1hiOb0iLJberEAAkU0JVvzOAr/D9ZUf+1MbUhexlNwn14iODT2Sqr2bj3oeHYd9Zqo4aJiQaDk+KaMq+UA==",
			},
		},
	}

	err := Verify(&r)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnauthenticated))
}
