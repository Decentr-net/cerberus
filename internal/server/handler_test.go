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
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"golang.org/x/net/context"

	"github.com/Decentr-net/go-api"
	apitest "github.com/Decentr-net/go-api/test"
	logging "github.com/Decentr-net/logrus/context"

	_ "github.com/Decentr-net/cerberus/internal/blockchain"
	"github.com/Decentr-net/cerberus/internal/entities"
	"github.com/Decentr-net/cerberus/internal/schema"
	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/internal/service/mock"
	"github.com/Decentr-net/cerberus/internal/throttler"
)

const testOwner = "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz"

var pdv = []byte(`{
    "version": "v1",
	"pdv": [
        {
			"timestamp": "2021-05-11T11:05:18Z",
			"source": {
			    "host": "decentr.net",
			    "path": "/"
			},
            "type": "cookie",
            "name": "my cookie",
            "value": "some value",
            "domain": "*",
            "hostOnly": true,
            "path": "*",
            "secure": true,
            "sameSite": "None",
            "expirationDate": 1861920000
        },
        {
			"timestamp": "2021-05-11T11:05:18Z",
			"source": {
			    "host": "decentr.net",
			    "path": "/"
			},
            "type": "cookie",
            "name": "my cookie 2",
            "value": "some value 2",
            "domain": "*",
            "hostOnly": true,
            "path": "*",
            "secure": true,
            "sameSite": "None",
            "expirationDate": 1861920000
        }
	]
}`)

var errSkip = errors.New("fictive error")

func newTestParameters(t *testing.T, method string, uri string, body []byte) (*bytes.Buffer, *httptest.ResponseRecorder, *http.Request) {
	l, w, r := apitest.NewAPITestParameters(method, uri, body)

	pk := secp256k1.PrivKey{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 0}

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
				var pdv schema.PDVWrapper
				require.NoError(t, json.Unmarshal(tc.reqBody, &pdv))

				srv.EXPECT().SavePDV(gomock.Any(), pdv, gomock.Any()).DoAndReturn(func(_ context.Context, _ schema.PDV, owner sdk.AccAddress) (uint64, *entities.PDVMeta, error) {
					if tc.err != nil {
						return 0, nil, tc.err
					}
					assert.Equal(t, testOwner, owner.String())
					return 1, &entities.PDVMeta{}, tc.err
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

			s := server{s: srv, maxPDVCount: 100, savePDVThrottler: throttler.New(5 * time.Minute)}
			router.Post("/v1/pdv", s.savePDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServerSavePDV_Throttler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mock.NewMockService(ctrl)

	s := server{s: srv, maxPDVCount: 100, savePDVThrottler: throttler.New(10 * time.Minute)}

	body := pdv

	var pdv schema.PDVWrapper
	require.NoError(t, json.Unmarshal(body, &pdv))
	srv.EXPECT().SavePDV(gomock.Any(), pdv, gomock.Any()).DoAndReturn(func(_ context.Context, _ schema.PDV, owner sdk.AccAddress) (uint64, *entities.PDVMeta, error) {
		assert.Equal(t, testOwner, owner.String())
		return 1, &entities.PDVMeta{}, nil
	})

	router := chi.NewRouter()

	do := func() *httptest.ResponseRecorder {
		_, w, r := newTestParameters(t, http.MethodPost, "v1/pdv", body)
		router.Post("/v1/pdv", s.savePDVHandler)
		router.ServeHTTP(w, r)
		return w
	}

	do()
	time.Sleep(1 * time.Second)
	w := do()

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
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
		f     func(_ context.Context, owner string, id uint64) (*entities.PDVMeta, error)
		rcode int
		rdata string
		rlog  string
	}{
		{
			name:  "success",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) (*entities.PDVMeta, error) {
				return &entities.PDVMeta{ObjectTypes: map[schema.Type]uint16{schema.PDVCookieType: 1}, Reward: sdk.NewDec(2)}, nil
			},
			rcode: http.StatusOK,
			rdata: `{"object_types":{"cookie": 1}, "reward": "2.000000000000000000"}`,
			rlog:  "",
		},
		{
			name:  "doesn't exists",
			owner: testOwner,
			id:    "1",
			f: func(_ context.Context, owner string, id uint64) (*entities.PDVMeta, error) {
				return nil, service.ErrNotFound
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
			f: func(_ context.Context, owner string, id uint64) (*entities.PDVMeta, error) {
				return nil, errors.New("test error")
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

			s := server{s: srv}
			router.Get("/v1/pdv/{owner}/{id}/meta", s.getPDVMetaHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.JSONEq(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServer_GetProfiles(t *testing.T) {
	tt := []struct {
		name         string
		url          string
		owner        []string
		f            func(_ context.Context, owner []string) ([]*entities.Profile, error)
		unauthorized bool
		rcode        int
		rdata        string
	}{
		{
			name:  "success",
			url:   "v1/profiles?address=decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz,decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
			owner: []string{testOwner, testOwner},
			f: func(_ context.Context, owner []string) ([]*entities.Profile, error) {
				return []*entities.Profile{
					{
						Address:   "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
						FirstName: "2",
						LastName:  "3",
						Emails:    []string{"email"},
						Bio:       "4",
						Avatar:    "5",
						Gender:    "6",
						Banned:    false,
						Birthday:  toTimePrt(time.Unix(1, 0)),
						CreatedAt: time.Unix(200000, 0),
					},
					{
						Address:   "decentr1u1slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
						FirstName: "22",
						LastName:  "23",
						Emails:    []string{"email"},
						Bio:       "24",
						Avatar:    "25",
						Gender:    "26",
						Banned:    false,
						Birthday:  toTimePrt(time.Unix(222210, 0)),
						CreatedAt: time.Unix(2200000, 0),
					},
					{
						Address:   "decentr1u1slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
						FirstName: "222",
						LastName:  "233",
						Emails:    []string{"email"},
						Bio:       "243",
						Avatar:    "253",
						Gender:    "263",
						Banned:    true,
						Birthday:  nil,
						CreatedAt: time.Unix(2300000, 0),
					},
				}, nil
			},
			rcode: http.StatusOK,
			rdata: `[
	{"address":"decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz","firstName":"2","lastName":"3","emails":["email"],"bio":"4","avatar":"5","gender":"6","banned":false, "birthday":"1970-01-01","createdAt":200000},
	{"address":"decentr1u1slwz3sje8j94ccpwlslflg0506yc8y2ylmtz","firstName":"22","lastName":"23","bio":"24","avatar":"25","gender":"26", "banned":false, "birthday":"1970-01-03","createdAt":2200000},
	{"address":"decentr1u1slwz3sje8j94ccpwlslflg0506yc8y2ylmtz","firstName":"222","lastName":"233","bio":"243","avatar":"253","gender":"263", "banned":true, "createdAt":2300000}
		]`,
		},
		{
			name:         "success_unauthorized",
			url:          "v1/profiles?address=decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz,decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
			owner:        []string{testOwner, testOwner},
			unauthorized: true,
			f: func(_ context.Context, owner []string) ([]*entities.Profile, error) {
				return []*entities.Profile{
					{
						Address:   "decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
						FirstName: "2",
						LastName:  "3",
						Emails:    []string{"email"},
						Bio:       "4",
						Avatar:    "5",
						Gender:    "6",
						Birthday:  toTimePrt(time.Unix(1, 0)),
						CreatedAt: time.Unix(200000, 0),
					},
					{
						Address:   "decentr1u1slwz3sje8j94ccpwlslflg0506yc8y2ylmtz",
						FirstName: "22",
						LastName:  "23",
						Emails:    []string{"email"},
						Bio:       "24",
						Avatar:    "25",
						Gender:    "26",
						Birthday:  toTimePrt(time.Unix(222210, 0)),
						CreatedAt: time.Unix(2200000, 0),
					},
				}, nil
			},
			rcode: http.StatusOK,
			rdata: `[
	{"address":"decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz","firstName":"2","lastName":"3","bio":"4","avatar":"5","gender":"6","birthday":"1970-01-01","createdAt":200000, "banned": false},
	{"address":"decentr1u1slwz3sje8j94ccpwlslflg0506yc8y2ylmtz","firstName":"22","lastName":"23","bio":"24","avatar":"25","gender":"26","birthday":"1970-01-03","createdAt":2200000, "banned": false}
		]`,
		},
		{
			name:  "invalid request",
			url:   "v1/profiles?address=1",
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"invalid address"}`,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, w, r := newTestParameters(t, http.MethodGet, tc.url, nil)

			if tc.unauthorized {
				r.Header.Del(api.SignatureHeader)
				r.Header.Del(api.PublicKeyHeader)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := mock.NewMockService(ctrl)

			if tc.f != nil {
				srv.EXPECT().GetProfiles(gomock.Any(), tc.owner).DoAndReturn(tc.f).Times(1)
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
			router.Get("/v1/profiles", s.getProfilesHandler)

			router.ServeHTTP(w, r)

			assert.Equal(t, tc.rcode, w.Code)
			assert.JSONEq(t, tc.rdata, w.Body.String())
		})
	}
}

func Test_getRewardsConfig(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mock.NewMockService(ctrl)

	srv.EXPECT().GetRewardsMap().Return(map[schema.Type]sdk.Dec{
		"cookie":  sdk.NewDecWithPrec(1, 6),
		"history": sdk.NewDecWithPrec(2, 6),
	})

	router := chi.NewRouter()

	s := server{s: srv}
	router.Get("/v1/configs/rewards", s.getRewardsConfigHandler)

	r := httptest.NewRequest(http.MethodGet, "http://localhost/v1/configs/rewards", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
		"cookie": "0.000001000000000000",
		"history": "0.000002000000000000"
	}`, w.Body.String())
}

func Test_getPDVRewardsPool(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	date := time.Date(2022, 1, 0, 1, 0, 0, 0, time.UTC)

	srv := mock.NewMockService(ctrl)
	srv.EXPECT().GetPDVTotalDelta(gomock.Any()).Return(sdk.NewDecWithPrec(3, 6), nil)
	srv.EXPECT().GetPDVRewardsNextDistributionDate(gomock.Any()).Return(date, nil)

	router := chi.NewRouter()

	s := server{s: srv}
	s.pdvRewardsPoolSize = sdk.NewDecWithPrec(15, 6)

	router.Get("/", s.getPDVRewardsPool)

	r := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
  "next_distribution_date": "2021-12-31T01:00:00Z",
  "size": "0.000015000000000000",
  "total_delta": "0.000003000000000000"
}`, w.Body.String())
}

func Test_getAccountPDVDelta(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	date := time.Date(2022, 1, 0, 1, 0, 0, 0, time.UTC)

	srv := mock.NewMockService(ctrl)
	srv.EXPECT().GetPDVTotalDelta(gomock.Any()).Return(sdk.NewDecWithPrec(3, 6), nil)
	srv.EXPECT().GetPDVRewardsNextDistributionDate(gomock.Any()).Return(date, nil)
	srv.EXPECT().GetPDVDelta(gomock.Any(), testOwner).Return(sdk.NewDecWithPrec(1, 6), nil)

	router := chi.NewRouter()

	s := server{s: srv}
	s.pdvRewardsPoolSize = sdk.NewDecWithPrec(15, 6)

	router.Get("/accounts/{owner}", s.getAccountPDVDelta)

	r := httptest.NewRequest(http.MethodGet, "http://localhost/accounts/"+testOwner, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
  "delta": "0.000001000000000000",
  "pool": {
    "next_distribution_date": "2021-12-31T01:00:00Z",
    "size": "0.000015000000000000",
    "total_delta": "0.000003000000000000"
  }
}`, w.Body.String())
}

func Test_savePDVHander_Amount(t *testing.T) {
	tt := []struct {
		name  string
		body  string
		valid bool
	}{
		{
			name:  "success",
			body:  string(pdv),
			valid: true,
		},
		{
			name: "profile",
			body: `{
                "version": "v1",
				"pdv": [
				{
		            "type": "profile",
		            "firstName": "John",
		            "lastName": "Dorian",
		            "emails": ["dev@decentr.xyz"],
		            "bio": "Just cool guy",
		            "gender": "",
		            "avatar": "http://john.dorian/avatar.png",
		            "birthday": "1993-01-20"
		        }
				]
			}`,
			valid: true,
		},
		{
			name: "too much",
			body: `{
                "version": "v1",
				"pdv": [
				{
					"timestamp": "2021-05-11T11:05:18Z",
					"type": "searchHistory",
					"engine": "decentr",
					"query": "the best crypto"
				},
				{
					"timestamp": "2021-05-11T11:05:18Z",
					"type": "searchHistory",
					"engine": "decentr",
					"query": "the best crypto"
				},
				{
					"timestamp": "2021-05-11T11:05:18Z",
					"type": "searchHistory",
					"engine": "decentr",
					"query": "the best crypto"
				},
				{
					"timestamp": "2021-05-11T11:05:18Z",
					"type": "searchHistory",
					"engine": "decentr",
					"query": "the best crypto"
				},
				{
					"timestamp": "2021-05-11T11:05:18Z",
					"type": "searchHistory",
					"engine": "decentr",
					"query": "the best crypto"
				}
				]
			}`,
			valid: false,
		},
		{
			name: "too little",
			body: `{
                "version": "v1",
				"pdv": [
				{
					"timestamp": "2021-05-11T11:05:18Z",
					"type": "searchHistory",
					"engine": "decentr",
					"query": "the best crypto"
				}
				]
			}`,
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, w, r := newTestParameters(t, http.MethodPost, "v1/pdv", []byte(tc.body))

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := mock.NewMockService(ctrl)

			srv.EXPECT().SavePDV(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			router := chi.NewRouter()

			s := server{s: srv, savePDVThrottler: throttler.New(1 * time.Minute), minPDVCount: 2, maxPDVCount: 4}
			router.Post("/v1/pdv", s.savePDVHandler)

			router.ServeHTTP(w, r)
			if tc.valid {
				require.Equal(t, http.StatusCreated, w.Code)
			} else {
				require.Equal(t, http.StatusBadRequest, w.Code)
			}
		})
	}
}

func toTimePrt(t time.Time) *time.Time {
	return &t
}
