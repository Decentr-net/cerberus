package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"golang.org/x/net/context"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

const testAddress = "e161f70a30964f22d7180bbf0fa7e87d1fa260e4-eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3a"

var pdv = []byte(`{
    "version": "v1",
	"pdv": {
	    "domain": "decentr.net",
	    "path": "/",
	    "user_agent": "mac",
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
	        },
	        {
	            "version": "v1",
	            "type": "cookie",
	            "name": "my cookie 2",
	            "value": "some value 2",
	            "domain": "*",
	            "host_only": true,
	            "path": "*",
	            "secure": true,
	            "same_site": "None",
	            "expiration_date": 1861920000
	        }
	    ]
	}
}`)

var errSkip = errors.New("fictive error")

func newTestParameters(t *testing.T, method string, uri string, body []byte) (*bytes.Buffer, *httptest.ResponseRecorder, *http.Request) {
	l := logrus.New()
	b := bytes.NewBufferString("")
	l.SetOutput(b)

	w := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), logCtxKey{}, l)
	r, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("http://localhost/%s", uri), bytes.NewReader(body))
	require.NoError(t, err)

	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	r.Header.Set("User-Agent", "mac")

	pk := secp256k1.PrivKeySecp256k1{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	require.NoError(t, api.Sign(r, pk))

	return b, w, r
}

func TestServer_SavePDVHandler(t *testing.T) {
	tt := []struct {
		name    string
		reqBody []byte
		err     error
		rcode   int
		rdata   string
		rlog    string
	}{
		{
			name:    "success",
			reqBody: pdv,
			err:     nil,
			rcode:   http.StatusCreated,
			rdata:   `{"address":"e161f70a30964f22d7180bbf0fa7e87d1fa260e4-57d274ad6d9226a499bc67cb67aa1770ad0f09df6014064035d77dd0d1ac2fb4"}`,
			rlog:    "",
		},
		{
			name:    "invalid request",
			reqBody: nil,
			err:     errSkip,
			rcode:   http.StatusBadRequest,
			rdata:   `{"error":"request is invalid: unexpected end of JSON input"}`,
			rlog:    "",
		},
		{
			name:    "invalid json",
			reqBody: []byte("some data"),
			err:     errSkip,
			rcode:   http.StatusBadRequest,
			rdata:   `{"error":"request is invalid: invalid character 's' looking for beginning of value"}`,
			rlog:    "",
		},
		{
			name: "invalid pdv",
			reqBody: []byte(`{
		   "version": "v1",
		   "pdv": {
		       "host": "decentr.net",
				"path": "",
		       "user_agent": "mac",
		       "data": []
		   }
		}`),
			err:   errSkip,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"pdv data is invalid"}`,
			rlog:  "",
		},
		{
			name:    "internal error",
			reqBody: pdv,
			err:     errors.New("test error"),
			rcode:   http.StatusInternalServerError,
			rdata:   `{"error":"internal error"}`,
			rlog:    "test error",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, w, r := newTestParameters(t, http.MethodPost, "v1/pdv", tc.reqBody)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := service.NewMockService(ctrl)

			if tc.err != errSkip {
				filepath, err := getPDVFilepath(r.Header.Get(api.PublicKeyHeader), tc.reqBody)
				require.NoError(t, err)

				srv.EXPECT().SavePDV(gomock.Any(), gomock.Any(), filepath).DoAndReturn(func(_ context.Context, d []byte, _ string) error {
					var pdv schema.PDV
					require.NoError(t, json.Unmarshal(tc.reqBody, &pdv))
					var spdv serverPDV
					require.NoError(t, json.Unmarshal(d, &spdv))

					assert.Equal(t, pdv, spdv.UserData)
					assert.Equal(t, metaPDVData{IP: "1.2.3.4", UserAgent: "mac"}, spdv.MetaData)

					return tc.err
				})
			}

			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log := logrus.New()
					log.SetOutput(b)
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), logCtxKey{}, log)))
				})
			})
			c, err := lru.NewARC(10)
			require.NoError(t, err)
			s := server{s: srv, pdvExistenceCache: c}
			router.Post("/v1/pdv", s.savePDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServer_ReceivePDVHandler(t *testing.T) {
	tt := []struct {
		name    string
		address string
		f       func(_ context.Context, address string) ([]byte, error)
		rcode   int
		rdata   string
		rlog    string
	}{
		{
			name:    "success",
			address: testAddress,
			f: func(_ context.Context, address string) ([]byte, error) {
				return []byte(`{"data":"cookie"}`), nil
			},
			rcode: http.StatusOK,
			rdata: `{"data":"cookie"}`,
			rlog:  "",
		},
		{
			name:    "invalid request",
			address: "adr",
			f:       nil,
			rcode:   http.StatusBadRequest,
			rdata:   `{"error":"invalid address"}`,
			rlog:    "",
		},
		{
			name:    "not found",
			address: testAddress,
			f: func(_ context.Context, address string) ([]byte, error) {
				return nil, service.ErrNotFound
			},
			rcode: http.StatusNotFound,
			rdata: fmt.Sprintf(`{"error":"PDV '%s' not found"}`, testAddress),
			rlog:  "",
		},
		{
			name:    "internal error",
			address: testAddress,
			f: func(_ context.Context, address string) ([]byte, error) {
				return nil, errors.New("test error")
			},
			rcode: http.StatusInternalServerError,
			rdata: `{"error":"internal error"}`,
			rlog:  "test error",
		},
		{
			name:    "forbidden error",
			address: "a" + testAddress[1:],
			f:       nil,
			rcode:   http.StatusForbidden,
			rdata:   `{"error":"access denied"}`,
			rlog:    "",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, w, r := newTestParameters(t, http.MethodGet, fmt.Sprintf("v1/pdv/%s", tc.address), nil)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := service.NewMockService(ctrl)

			if tc.f != nil {
				srv.EXPECT().ReceivePDV(gomock.Any(), tc.address).DoAndReturn(tc.f)
			}

			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log := logrus.New()
					log.SetOutput(b)
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), logCtxKey{}, log)))
				})
			})
			s := server{s: srv}
			router.Get("/v1/pdv/{address}", s.receivePDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServer_DoesPDVExistHandler(t *testing.T) {
	tt := []struct {
		name    string
		address string
		f       func(_ context.Context, address string) (bool, error)
		rcode   int
		rdata   string
		rlog    string
	}{
		{
			name:    "exists",
			address: testAddress,
			f: func(_ context.Context, address string) (bool, error) {
				return true, nil
			},
			rcode: http.StatusOK,
			rlog:  "",
		},
		{
			name:    "doesn't exists",
			address: testAddress,
			f: func(_ context.Context, address string) (bool, error) {
				return false, nil
			},
			rcode: http.StatusNotFound,
			rlog:  "",
		},
		{
			name:    "invalid request",
			address: "invalid",
			f:       nil,
			rcode:   http.StatusBadRequest,
			rdata:   `{"error":"invalid address"}`,
			rlog:    "",
		},
		{
			name:    "internal error",
			address: testAddress,
			f: func(_ context.Context, address string) (bool, error) {
				return false, errors.New("test error")
			},
			rcode: http.StatusInternalServerError,
			rdata: `{"error":"internal error"}`,
			rlog:  "test error",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, w, r := newTestParameters(t, http.MethodHead, fmt.Sprintf("v1/pdv/%s", tc.address), nil)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := service.NewMockService(ctrl)

			if tc.f != nil {
				srv.EXPECT().DoesPDVExist(gomock.Any(), tc.address).DoAndReturn(tc.f)
			}

			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log := logrus.New()
					log.SetOutput(b)
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), logCtxKey{}, log)))
				})
			})
			c, err := lru.NewARC(10)
			require.NoError(t, err)
			s := server{s: srv, pdvExistenceCache: c}
			router.Head("/v1/pdv/{address}", s.doesPDVExistHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}
