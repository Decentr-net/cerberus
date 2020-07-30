package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

var testAddress = "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3a"

func startServer(t *testing.T, c int, d string, expect string, expectQuery string) int {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(b), expect)
		assert.Equal(t, expectQuery, r.URL.Query().Encode())

		w.WriteHeader(c)
		w.Write([]byte(d))
	}))

	return l.Addr().(*net.TCPAddr).Port
}

func TestClient_SendPDV(t *testing.T) {
	tt := []struct {
		name     string
		code     int
		response string

		address string
		err     string
	}{
		{
			name:     "success",
			code:     http.StatusCreated,
			response: fmt.Sprintf(`{"address":"%s"}`, testAddress),

			address: testAddress,
			err:     "",
		},
		{
			name:     "internal error",
			code:     http.StatusUnauthorized,
			response: `{"error":"something went wrong"}`,

			address: "",
			err:     "failed to make SendPDV request: request failed: something went wrong",
		},
		{
			name:     "bad request",
			code:     http.StatusBadRequest,
			response: `{"error":"something went wrong"}`,

			address: "",
			err:     "failed to make SendPDV request: invalid request",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, `"data":"ZGF0YQ=="`, "")

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			address, err := c.SendPDV(context.Background(), []byte("data"))
			assert.Equal(t, tc.address, address)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}

func TestClient_DoesPDVExist(t *testing.T) {
	tt := []struct {
		name     string
		code     int
		response string

		exists bool
		err    string
	}{
		{
			name:     "success",
			code:     http.StatusOK,
			response: `{"exists":true}`,

			exists: true,
			err:    "",
		},
		{
			name:     "unknown error",
			code:     http.StatusInternalServerError,
			response: ``,

			exists: false,
			err:    "failed to make DoesPDVExist request: request failed with status 500",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, "", fmt.Sprintf(`address=%s`, url.QueryEscape(testAddress)))

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			exists, err := c.DoesPDVExist(context.Background(), testAddress)
			assert.Equal(t, tc.exists, exists)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}

func TestClient_ReceivePDV(t *testing.T) {
	tt := []struct {
		name     string
		code     int
		response string

		data []byte
		err  string
	}{
		{
			name:     "success",
			code:     http.StatusOK,
			response: `{"data":"ZGF0YQ=="}`,

			data: []byte("data"),
			err:  "",
		},
		{
			name:     "not found",
			code:     http.StatusNotFound,
			response: `{"error":"not found"}`,

			data: nil,
			err:  "failed to make ReceivePDV request: not found",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, fmt.Sprintf(`"address":"%s"`, testAddress), "")

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			data, err := c.ReceivePDV(context.Background(), testAddress)
			assert.Equal(t, tc.data, data)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}

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

	_, err := Verify(&r)
	assert.NoError(t, err, "can not verify signed request")

	r.unexported, r.ExcludedFromJSONMarshal, r.ExcludedPtr = 1, 1, &r.StringData
	_, err = Verify(&r)
	assert.NoError(t, err, "digest changed")

	r.StringData = "new_string"
	_, err = Verify(&r)
	assert.True(t, errors.Is(err, ErrNotVerified), "digest not changed")
}
