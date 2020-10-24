package api

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto/secp256k1"
)

var testSignature = "f8f173f2de49a6ce040fa963ff510debeadf118c8972ba1ee19310eae3dd616931b4ffabb351ce8e38ce6984dfadb5aae8e2be6d7a029346be6c8a50ace6a56f"

func TestVerify(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))

		assert.NoError(t, Verify(r))
	})

	t.Run("invalid key", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))

		r.Header.Set(PublicKeyHeader, "invalid")

		err = Verify(r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidPublicKey))
	})

	t.Run("invalid key 2", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))

		r.Header.Set(PublicKeyHeader, testSignature)

		err = Verify(r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidPublicKey))
	})

	t.Run("invalid signature", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))

		r.Header.Set(SignatureHeader, "invalid")

		err = Verify(r)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidSignature))
	})

	t.Run("wrong signature", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))
		r.Header.Set(SignatureHeader, testSignature)

		err = Verify(r)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotVerified))
	})
}

func TestGetMessageToSign(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "https://localhost/path/to/file", bytes.NewBufferString("some/body"))
	require.NoError(t, err)

	b, err := GetMessageToSign(r)
	require.NoError(t, err)
	assert.Equal(t, "some/body/path/to/file", string(b))
}

func TestSign(t *testing.T) {
	key := secp256k1.PrivKeySecp256k1{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	tt := []struct {
		name string
		path string
		body []byte

		signature string
	}{
		{
			name: "body",
			path: "",
			body: []byte{1, 2, 3, 2, 1},

			signature: "9ebf708f5d0eeda13c27c2ee2324bc7c9ce62e404bba5febfb12408fb5026eca46a588ea7aac181f47bbeb261b899e411d7dd1a6e5a71cd832bb7bd6127868d9",
		},
		{
			name: "path",
			path: "file",
			body: nil,

			signature: "08618c88842a20c360a3a52a996707bcb235dbc0a85473989d0e2d3d99ffb2564bec3f8b42aeb2c91cbf64b856ee6c534558c32ff65cb352ed1f2ef96b2d2478",
		},
		{
			name: "path+body",
			path: "path",
			body: []byte{1, 2, 3, 2, 1},

			signature: "efa3aba44216d69a6bae488f02b334c292f0449247e47fd8ef8c3cb6bb43adbe406c3bc05cbb7009c2f3c69cf8ac5df9d190bba07c952057f62d314f85fa6417",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost/%s", tc.path), bytes.NewReader(tc.body))
			require.NoError(t, err)

			require.NoError(t, Sign(r, key))

			assert.Equal(t, hex.EncodeToString(key.PubKey().Bytes()[5:]), r.Header.Get(PublicKeyHeader))
			assert.Equal(t, tc.signature, r.Header.Get(SignatureHeader))
		})
	}
}
