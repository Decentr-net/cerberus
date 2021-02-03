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

const testOwner = "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz"
const testID = 1

var rawPDV = []byte(`{
    "version": "v1",
	"pdv": [
		{
		    "domain": "decentr.net",
		    "path": "/",
			"data": [
		        {
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
				&schema.PDVDataCookie{
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

		id  uint64
		err string
	}{
		{
			name:     "success",
			code:     http.StatusCreated,
			response: []byte(fmt.Sprintf(`{"id":%d}`, testID)),

			id:  testID,
			err: "",
		},
		{
			name:     "internal error",
			code:     http.StatusUnauthorized,
			response: []byte(`{"error":"something went wrong"}`),

			id:  0,
			err: "failed to make SavePDV request: request failed: something went wrong",
		},
		{
			name:     "bad request",
			code:     http.StatusBadRequest,
			response: []byte(`{"error":"something went wrong"}`),

			id:  0,
			err: "failed to make SavePDV request: invalid request",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, "/v1/pdv", rawPDV)

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			id, err := c.SavePDV(context.Background(), pdv)
			assert.Equal(t, tc.id, id)

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

			p := startServer(t, tc.code, tc.response, fmt.Sprintf("/v1/pdv/%s/%d/meta", testOwner, testID), nil)

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			m, err := c.GetPDVMeta(context.Background(), testOwner, testID)
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

			p := startServer(t, tc.code, tc.response, fmt.Sprintf("/v1/pdv/%s/%d", testOwner, testID), nil)

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			data, err := c.ReceivePDV(context.Background(), testOwner, testID)
			assert.Equal(t, tc.data, data)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}

func TestClient_ListPDV(t *testing.T) {
	tt := []struct {
		name     string
		code     int
		response []byte

		list []uint64
		err  string
	}{
		{
			name:     "success",
			code:     http.StatusOK,
			response: []byte(`[1,2,3,4]`),

			list: []uint64{1, 2, 3, 4},
			err:  "",
		},
		{
			name:     "error",
			code:     http.StatusInternalServerError,
			response: []byte(`{"error":"internal error"}`),

			list: nil,
			err:  "failed to make ListPDV request: request failed: internal error",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := startServer(t, tc.code, tc.response, fmt.Sprintf("/v1/pdv/%s?from=0&limit=1000", testOwner), nil)

			c := NewClient(fmt.Sprintf("http://localhost:%d", p), secp256k1.GenPrivKey()).(*client)

			list, err := c.ListPDV(context.Background(), testOwner, 0, 1000)
			assert.Equal(t, tc.list, list)

			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
		})
	}
}
