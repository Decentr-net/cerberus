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

		d, err := Verify(r)
		assert.NoError(t, err)
		assert.NotNil(t, d)
	})

	t.Run("invalid key", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))

		r.Header.Set(PublicKeyHeader, "invalid")

		_, err = Verify(r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidPublicKey))
	})

	t.Run("invalid key 2", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))

		r.Header.Set(PublicKeyHeader, testSignature)

		_, err = Verify(r)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidPublicKey))
	})

	t.Run("invalid signature", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))

		r.Header.Set(SignatureHeader, "invalid")

		_, err = Verify(r)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidSignature))
	})

	t.Run("wrong signature", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "https://localhost/file", bytes.NewBufferString("some"))
		require.NoError(t, err)

		key := secp256k1.GenPrivKey()
		require.NoError(t, Sign(r, key))
		r.Header.Set(SignatureHeader, testSignature)

		_, err = Verify(r)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotVerified))
	})
}

func TestDigest(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "https://localhost/path/to/file", bytes.NewBufferString("some/body"))
	require.NoError(t, err)

	b, err := Digest(r)
	require.NoError(t, err)
	assert.Equal(t, "1d6fb0a9196ef05aabda9d9c2461e5eb152cbb81911d133a80f9bfdf8a5181ac", hex.EncodeToString(b))
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

			signature: "dea757c817bf4c38ce43f8234663f20feb144768906f2dec0d9f01d3ed14fd7f43df99efb094d46ed26109a682beeab1259bf6be1d35a7495d6be04846808fef",
		},
		{
			name: "path",
			path: "file",
			body: nil,

			signature: "1eefbc1f355f00e27c14362119614e231d128d040dd46140571ead9f82392e0b60d60d7afb888ac19dd4a732a7756ed2f04e61a8a7266c095c4f8158bcad5172",
		},
		{
			name: "path+body",
			path: "path",
			body: []byte{1, 2, 3, 2, 1},

			signature: "2632c0fc23c21ab5e432a1607815f13bcd815a6a851a0541c508131101ecd92417ab40e955982bee85a73b2bb344bb27e30933bd5da44be1c75a55af49b49762",
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
