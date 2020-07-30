package api

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

var testSignature = "f8f173f2de49a6ce040fa963ff510debeadf118c8972ba1ee19310eae3dd616931b4ffabb351ce8e38ce6984dfadb5aae8e2be6d7a029346be6c8a50ace6a56f"

func TestVerify(t *testing.T) {
	type TestAuthRequest struct {
		AuthRequest
		FilePath string
	}

	t.Run("valid", func(t *testing.T) {
		r := TestAuthRequest{
			FilePath: "somefile",
		}

		key := secp256k1.GenPrivKey()
		digest, err := Digest(r)
		require.NoError(t, err)
		require.NotNil(t, digest)
		signature, err := key.Sign(digest)
		require.NoError(t, err)
		require.NotNil(t, signature)

		pub := key.PubKey()

		r.Signature = Signature{
			PublicKey: hex.EncodeToString(pub.Bytes()),
			Signature: hex.EncodeToString(signature),
		}

		d, err := Verify(&r)
		assert.NoError(t, err)
		assert.NotNil(t, d)
	})

	t.Run("invalid key", func(t *testing.T) {
		r := TestAuthRequest{
			AuthRequest: AuthRequest{
				Signature: Signature{
					PublicKey: "wrong key",
					Signature: testSignature,
				},
			},
		}

		_, err := Verify(&r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidPublicKey))
	})

	t.Run("invalid key 2", func(t *testing.T) {
		r := TestAuthRequest{
			AuthRequest: AuthRequest{
				Signature: Signature{
					PublicKey: testSignature,
					Signature: testSignature,
				},
			},
		}

		_, err := Verify(&r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidPublicKey))
	})

	t.Run("invalid signature", func(t *testing.T) {
		r := TestAuthRequest{
			AuthRequest: AuthRequest{
				Signature: Signature{
					PublicKey: hex.EncodeToString(secp256k1.GenPrivKey().PubKey().Bytes()),
					Signature: "not base64",
				},
			},
		}

		_, err := Verify(&r)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidSignature))
	})

	t.Run("wrong signature", func(t *testing.T) {
		r := TestAuthRequest{
			FilePath: "myfile",
			AuthRequest: AuthRequest{
				Signature: Signature{
					PublicKey: hex.EncodeToString(secp256k1.GenPrivKey().PubKey().Bytes()),
					Signature: testSignature[1:] + "a",
				},
			},
		}

		_, err := Verify(&r)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotVerified))
	})
}

func TestDigest(t *testing.T) {
	r := struct {
		AuthRequest

		StringData              string
		IntData                 int64
		SliceData               []float64
		unexported              int
		ExcludedFromJSONMarshal int     `json:"-"`
		ExcludedPtr             *string `json:"-"`
	}{
		StringData:              "string",
		IntData:                 42,
		SliceData:               []float64{1.1, 2.2, 3.3},
		unexported:              42,
		ExcludedFromJSONMarshal: 42,
		ExcludedPtr:             nil,
	}

	b, err := Digest(r)
	require.NoError(t, err)
	assert.Equal(t, "8a16f5cfc09fce0a8852aa3291c8ae3c2c35d5f0345d1a6a9a073fa77106c010", hex.EncodeToString(b))
}
