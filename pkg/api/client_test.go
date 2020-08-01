package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

var testAddress = "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6-eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3a"
var pdv = []byte(`{
    "version": "v1",
    "pdv": {
        "ip": "1.1.1.1",
        "user_agent": "mac",
        "data": [
            {
                "version": "v1",
                "type": "cookie",
                "name": "my cookie",
                "value": "some value",
                "expires": "some date",
                "max_age": 1234,
                "path": "path",
                "domain": "domain"
            },
            {
                "version": "v1",
                "type": "cookie",
                "name": "my cookie",
                "value": "some value",
                "expires": "some date",
                "max_age": 1234,
                "path": "path",
                "domain": "domain"
            }
        ]
    }
}`)

func startServer(t *testing.T, c int, d []byte, path string, data []byte) int {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.RequestURI)

		if data == nil {
			assert.Equal(t, http.NoBody, r.Body)
		} else {
			b, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, data, b)
		}

		w.WriteHeader(c)
		w.Write(d)
	}))

	return l.Addr().(*net.TCPAddr).Port
}

func TestClient_SendPDV(t *testing.T) {
	tt := []struct {
		name     string
		code     int
		response []byte

		address string
		err     string
	}{
		{
			name:     "success",
			code:     http.StatusCreated,
			response: []byte(fmt.Sprintf(`{"address":"%s"}`, testAddress)),

			address: testAddress,
			err:     "",
		},
		{
			name:     "internal error",
			code:     http.StatusUnauthorized,
			response: []byte(`{"error":"something went wrong"}`),

			address: "",
			err:     "failed to make SendPDV request: request failed: something went wrong",
		},
		{
			name:     "bad request",
			code:     http.StatusBadRequest,
			response: []byte(`{"error":"something went wrong"}`),

			address: "",
			err:     "failed to make SendPDV request: invalid request",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, "/v1/pdv", pdv)

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			address, err := c.SendPDV(context.Background(), pdv)
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
		name string
		code int

		exists bool
		err    string
	}{
		{
			name: "success",
			code: http.StatusOK,

			exists: true,
			err:    "",
		},
		{
			name: "false",
			code: http.StatusNotFound,

			exists: false,
			err:    "",
		},
		{
			name: "unknown error",
			code: http.StatusInternalServerError,

			exists: false,
			err:    "failed to make DoesPDVExist request: request failed with status 500",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, nil, fmt.Sprintf("/v1/pdv/%s", testAddress), nil)

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
		response []byte

		data json.RawMessage
		err  string
	}{
		{
			name:     "success",
			code:     http.StatusOK,
			response: []byte(`{"json":"yes"}`),

			data: json.RawMessage(`{"json":"yes"}`),
			err:  "",
		},
		{
			name:     "not found",
			code:     http.StatusNotFound,
			response: []byte(`{"error":"not found"}`),

			data: nil,
			err:  "failed to make ReceivePDV request: not found",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, fmt.Sprintf("/v1/pdv/%s", testAddress), nil)

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
