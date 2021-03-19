package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"golang.org/x/net/context"

	apitest "github.com/Decentr-net/go-api/test"
	logging "github.com/Decentr-net/logrus/context"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/internal/service/mock"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

const testOwner = "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz"

var pdv = []byte(`{
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
	]
}`)

var errSkip = errors.New("fictive error")

func newTestParameters(t *testing.T, method string, uri string, body []byte) (*bytes.Buffer, *httptest.ResponseRecorder, *http.Request) {
	l, w, r := apitest.NewAPITestParameters(method, uri, body)
	pk := secp256k1.PrivKeySecp256k1{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	require.NoError(t, api.Sign(r, pk))

	return l, w, r
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
			rdata:   `{"id":1}`,
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
			name:    "invalid pdv",
			reqBody: []byte(`{"version": "v1"}`),
			err:     errSkip,
			rcode:   http.StatusBadRequest,
			rdata:   `{"error":"pdv data is invalid"}`,
			rlog:    "",
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

			srv := mock.NewMockService(ctrl)

			if tc.err != errSkip {
				var pdv schema.PDV
				require.NoError(t, json.Unmarshal(tc.reqBody, &pdv))

				srv.EXPECT().SavePDV(gomock.Any(), pdv, gomock.Any()).DoAndReturn(func(_ context.Context, _ schema.PDV, owner types.AccAddress) (uint64, api.PDVMeta, error) {
					assert.Equal(t, testOwner, owner.String())
					return 1, api.PDVMeta{}, tc.err
				})
			}

			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log := logrus.New()
					log.SetOutput(b)
					next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), log)))
				})
			})
			c, err := lru.NewARC(10)
			require.NoError(t, err)
			s := server{s: srv, pdvMetaCache: c, maxPDVCount: 100}
			router.Post("/v1/pdv", s.savePDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServer_ListPDVHandler(t *testing.T) {
	tt := []struct {
		name  string
		owner string
		from  uint64
		limit uint16
		list  []uint64
		err   error

		rcode int
		rdata string
		rlog  string
	}{
		{
			name:  "success",
			owner: testOwner,
			list:  []uint64{1, 2, 3, 4},
			err:   nil,

			rcode: http.StatusOK,
			rdata: `[1,2,3,4]`,
			rlog:  "",
		},
		{
			name:  "success_params",
			owner: testOwner,
			list:  []uint64{1, 2, 3, 4},
			from:  5,
			limit: 200,
			err:   nil,

			rcode: http.StatusOK,
			rdata: `[1,2,3,4]`,
			rlog:  "",
		},
		{
			name:  "invalid request",
			owner: "adr",

			rcode: http.StatusBadRequest,
			rdata: `{"error":"invalid owner"}`,
			rlog:  "",
		},
		{
			name:  "invalid request params",
			owner: testOwner,
			limit: 1001,

			rcode: http.StatusBadRequest,
			rdata: `{"error":"invalid limit"}`,
			rlog:  "",
		},
		{
			name:  "internal error",
			owner: testOwner,
			list:  nil,
			err:   errors.New("test error"),
			rcode: http.StatusInternalServerError,
			rdata: `{"error":"internal error"}`,
			rlog:  "test error",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, w, r := newTestParameters(t, http.MethodGet, fmt.Sprintf("v1/pdv/%s", tc.owner), nil)
			q := r.URL.Query()
			if tc.from != 0 {
				q.Set("from", strconv.FormatUint(tc.from, 10))
			}
			if tc.limit != 0 {
				q.Set("limit", strconv.FormatUint(uint64(tc.limit), 10))
			}
			r.URL.RawQuery = q.Encode()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := mock.NewMockService(ctrl)

			if tc.rcode != http.StatusBadRequest {
				limit := defaultLimit
				if tc.limit != 0 {
					limit = uint64(tc.limit)
				}
				srv.EXPECT().ListPDV(gomock.Any(), tc.owner, tc.from, uint16(limit)).Return(tc.list, tc.err)
			}

			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log := logrus.New()
					log.SetOutput(b)
					next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), log)))
				})
			})
			s := server{s: srv}
			router.Get("/v1/pdv/{owner}", s.listPDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServer_ReceivePDVHandler(t *testing.T) {
	tt := []struct {
		name  string
		owner string
		id    string
		f     func(_ context.Context, owner string, id uint64) ([]byte, error)
		rcode int
		rdata string
		rlog  string
	}{
		{
			name:  "success",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) ([]byte, error) {
				return []byte(`{"data":"cookie"}`), nil
			},
			rcode: http.StatusOK,
			rdata: `{"data":"cookie"}`,
			rlog:  "",
		},
		{
			name:  "invalid request",
			owner: "adr",
			id:    "1",
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"invalid owner"}`,
			rlog:  "",
		},
		{
			name:  "invalid request #2",
			owner: testOwner,
			id:    "1s",
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"invalid id"}`,
			rlog:  "",
		},
		{
			name:  "not found",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) ([]byte, error) {
				return nil, service.ErrNotFound
			},
			rcode: http.StatusNotFound,
			rdata: fmt.Sprintf(`{"error":"PDV '%s' not found"}`, "1"),
			rlog:  "",
		},
		{
			name:  "internal error",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) ([]byte, error) {
				return nil, errors.New("test error")
			},
			rcode: http.StatusInternalServerError,
			rdata: `{"error":"internal error"}`,
			rlog:  "test error",
		},
		{
			name:  "forbidden error",
			owner: "decentr1ltx6yymrs8eq4nmnhzfzxj6tspjuymh8mgd6gz",
			id:    "1",
			f:     nil,
			rcode: http.StatusForbidden,
			rdata: `{"error":"access denied"}`,
			rlog:  "",
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, w, r := newTestParameters(t, http.MethodGet, fmt.Sprintf("v1/pdv/%s/%s", tc.owner, tc.id), nil)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := mock.NewMockService(ctrl)

			if tc.f != nil {
				id, err := strconv.ParseUint(tc.id, 10, 64)
				require.NoError(t, err)
				srv.EXPECT().ReceivePDV(gomock.Any(), tc.owner, id).DoAndReturn(tc.f)
			}

			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log := logrus.New()
					log.SetOutput(b)
					next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), log)))
				})
			})
			s := server{s: srv}
			router.Get("/v1/pdv/{owner}/{id}", s.getPDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServer_GetPDVMeta(t *testing.T) {
	tt := []struct {
		name  string
		owner string
		id    string
		f     func(_ context.Context, owner string, id uint64) (api.PDVMeta, error)
		rcode int
		rdata string
		rlog  string
	}{
		{
			name:  "success",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) (api.PDVMeta, error) {
				return api.PDVMeta{ObjectTypes: map[schema.PDVType]uint16{schema.PDVCookieType: 1}, Reward: 2}, nil
			},
			rcode: http.StatusOK,
			rdata: `{"object_types":{"cookie": 1}, "reward": 2}`,
			rlog:  "",
		},
		{
			name:  "doesn't exists",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) (api.PDVMeta, error) {
				return api.PDVMeta{}, service.ErrNotFound
			},
			rcode: http.StatusNotFound,
			rdata: fmt.Sprintf(`{"error":"PDV '%s' not found"}`, "1"),
			rlog:  "",
		},
		{
			name:  "invalid request",
			owner: "inv",
			id:    "1",
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"invalid address"}`,
			rlog:  "",
		},
		{
			name:  "invalid request #2",
			owner: testOwner,
			id:    "1s",
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"invalid id"}`,
			rlog:  "",
		},
		{
			name:  "internal error",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) (api.PDVMeta, error) {
				return api.PDVMeta{}, errors.New("test error")
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

			b, w, r := newTestParameters(t, http.MethodGet, fmt.Sprintf("v1/pdv/%s/%s/meta", tc.owner, tc.id), nil)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := mock.NewMockService(ctrl)

			if tc.f != nil {
				id, err := strconv.ParseUint(tc.id, 10, 64)
				require.NoError(t, err)
				srv.EXPECT().GetPDVMeta(gomock.Any(), tc.owner, id).DoAndReturn(tc.f).Times(1)
			}

			router := chi.NewRouter()
			router.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					log := logrus.New()
					log.SetOutput(b)
					next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), log)))
				})
			})
			c, err := lru.NewARC(10)
			require.NoError(t, err)
			s := server{s: srv, pdvMetaCache: c}
			router.Get("/v1/pdv/{owner}/{id}/meta", s.getPDVMetaHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.JSONEq(t, tc.rdata, w.Body.String())

			// check cache
			if tc.rcode == http.StatusOK {
				_, w, r := newTestParameters(t, http.MethodGet, fmt.Sprintf("v1/pdv/%s/%s/meta", tc.owner, tc.id), nil)

				router.ServeHTTP(w, r)

				assert.Equal(t, tc.rcode, w.Code)
				assert.JSONEq(t, tc.rdata, w.Body.String())
			}
		})
	}
}
