package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

func TestClient_signRequest(t *testing.T) {
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

	c := NewClient("", secp256k1.GenPrivKey()).(*client)

	require.NoError(t, c.signRequest(&r))
	assert.NotEmpty(t, r.AuthRequest.Signature.Signature)
	assert.NotEmpty(t, r.AuthRequest.Signature.PublicKey)

	assert.NoError(t, Verify(&r), "can not verify signed request")

	r.unexported, r.ExcludedFromJSONMarshal, r.ExcludedPtr = 1, 1, &r.StringData
	assert.NoError(t, Verify(&r), "digest changed")

	r.StringData = "new_string"
	assert.True(t, errors.Is(Verify(&r), ErrNotVerified), "digest not changed")
}
