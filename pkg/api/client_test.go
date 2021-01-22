package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/Decentr-net/cerberus/pkg/schema"
)

var testAddress = "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6-eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3a"

var rawPDV = []byte(`{
    "version": "v1",
	"pdv": [
		{
		    "domain": "decentr.net",
		    "path": "/",
			"data": [
		        {
		            "version": "v1",
		            "type": "cookie",
		            "name": "my cookie",
		            "value": "some value",
		            "domain": "*",
		            "host_only": true,
		            "path": "*",
		            "secure": true,
		            "same_site": "None",
		            "expiration_date": 1861920000
		        }
		    ]
		}
	]
}`)

var pdv = schema.PDV{
	Version: schema.PDVV1,
	PDV: []schema.PDVObject{
		&schema.PDVObjectV1{
			PDVObjectMetaV1: schema.PDVObjectMetaV1{
				Host: "decentr.net",
				Path: "/",
			},
			Data: []schema.PDVData{
				&schema.PDVDataCookieV1{
					Name:           "my cookie",
					Value:          "some value",
					Domain:         "*",
					HostOnly:       true,
					Path:           "*",
					Secure:         true,
					SameSite:       "None",
					ExpirationDate: 1861920000,
				},
			},
		},
	},
}

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
			assert.JSONEq(t, string(data), string(b))
		}

		w.WriteHeader(c)
		w.Write(d)
	}))

	return l.Addr().(*net.TCPAddr).Port
}

func TestClient_SavePDV(t *testing.T) {
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
			err:     "failed to make SavePDV request: request failed: something went wrong",
		},
		{
			name:     "bad request",
			code:     http.StatusBadRequest,
			response: []byte(`{"error":"something went wrong"}`),

			address: "",
			err:     "failed to make SavePDV request: invalid request",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, "/v1/pdv", rawPDV)

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			address, err := c.SavePDV(context.Background(), pdv)
			assert.Equal(t, tc.address, address)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}

func TestClient_GetPDVMeta(t *testing.T) {
	tt := []struct {
		name     string
		code     int
		response []byte

		meta PDVMeta
		err  string
	}{
		{
			name: "success",
			code: http.StatusOK,
			response: []byte(`{
	"object_types": {
		"cookie": 5,
		"login_cookie": 1
	}
}
`),

			meta: PDVMeta{ObjectTypes: map[schema.PDVType]uint16{
				schema.PDVCookieType:      5,
				schema.PDVLoginCookieType: 1,
			}},
			err: "",
		},
		{
			name: "false",
			code: http.StatusNotFound,

			err: "failed to make GetPDVMeta request: not found",
		},
		{
			name: "unknown error",
			code: http.StatusInternalServerError,

			err: "failed to make GetPDVMeta request: request failed with status 500",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, fmt.Sprintf("/v1/pdv/%s/meta", testAddress), nil)

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			m, err := c.GetPDVMeta(context.Background(), testAddress)
			assert.Equal(t, tc.meta, m)

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

		data schema.PDV
		err  string
	}{
		{
			name:     "success",
			code:     http.StatusOK,
			response: rawPDV,

			data: pdv,
			err:  "",
		},
		{
			name:     "not found",
			code:     http.StatusNotFound,
			response: []byte(`{"error":"not found"}`),

			data: schema.PDV{},
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
