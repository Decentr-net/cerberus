package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"golang.org/x/net/context"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
)

var wrongSignature = api.Signature{
	PublicKey: "eb5ae987210385f66f360ffc57607fc69a7b5fbd06f92841db02521853f5ebdc7bc983a35901",
	Signature: "f8f173f2de49a6ce040fa963ff510debeadf118c8972ba1ee19310eae3dd616931b4ffabb351ce8e38ce6984dfadb5aae8e2be6d7a029346be6c8a50ace6a56f",
}
var errSkip = errors.New("fictive error")

const invalidJSON = "invalid json"

func newTestParameters(t *testing.T) (*bytes.Buffer, *httptest.ResponseRecorder, *http.Request) {
	l := logrus.New()
	b := bytes.NewBufferString("")
	l.SetOutput(b)

	w := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), logCtxKey{}, l)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "", nil)
	require.NoError(t, err)

	return b, w, r
}

func getSignature(t *testing.T, r interface{}) api.Signature {
	d, err := api.Digest(r)
	require.NoError(t, err)

	pk := secp256k1.PrivKeySecp256k1{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}

	sign, err := pk.Sign(d)
	require.NoError(t, err)

	return api.Signature{
		PublicKey: hex.EncodeToString(pk.PubKey().Bytes()),
		Signature: hex.EncodeToString(sign),
	}
}

func pathRequest(t *testing.T, r *http.Request, endpoint string, d interface{}, s *api.Signature) {
	var err error
	r.URL, err = url.Parse(fmt.Sprintf("http://localhost%s", endpoint))
	require.NoError(t, err)

	// test incorrect signature
	if s.Signature == "" {
		*s = getSignature(t, d)
	}

	b, err := json.Marshal(d)
	require.NoError(t, err)

	r.Body = ioutil.NopCloser(bytes.NewReader(b))
}

func TestServer_SendPDVHandler(t *testing.T) {
	getFilename := func(r *api.SendPDVRequest) string {
		d, err := api.Digest(r)
		require.NoError(t, err)

		return fmt.Sprintf("%s/%s", r.Signature.PublicKey, hex.EncodeToString(d))
	}

	tt := []struct {
		name  string
		req   *api.SendPDVRequest
		err   error
		rcode int
		rdata string
		rlog  string
	}{
		{
			name: "success",
			req: &api.SendPDVRequest{
				Data: []byte("some data"),
			},
			err:   nil,
			rcode: http.StatusCreated,
			rdata: `{"address":"eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/3d7f924ef9c2be7af75a9e3266817a498c06de00461d5033d1825f536c4aad6f"}`,
			rlog:  "",
		},
		{
			name:  "invalid request",
			req:   &api.SendPDVRequest{},
			err:   errSkip,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"request is invalid"}`,
			rlog:  "",
		},
		{
			name: "invalid json",
			req: &api.SendPDVRequest{
				Data:        []byte(invalidJSON),
				AuthRequest: api.AuthRequest{Signature: wrongSignature},
			},
			err:   errSkip,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"failed to decode json"}`,
			rlog:  "",
		},
		{
			name: "internal error",
			req: &api.SendPDVRequest{
				Data: []byte("some data"),
			},
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

			b, w, r := newTestParameters(t)
			pathRequest(t, r, api.SendPDVEndpoint, tc.req, &tc.req.Signature)

			if string(tc.req.Data) == invalidJSON {
				r.Body = ioutil.NopCloser(bytes.NewBufferString(invalidJSON))
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := service.NewMockService(ctrl)

			if tc.err != errSkip {
				srv.EXPECT().SendPDV(gomock.Any(), tc.req.Data, getFilename(tc.req)).Return(tc.err)
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
			router.Post(api.SendPDVEndpoint, s.sendPDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}

func TestServer_DoesPDVExistHandler(t *testing.T) {
	tt := []struct {
		name  string
		req   *api.DoesPDVExistRequest
		f     func(_ context.Context, address string) (bool, error)
		rcode int
		rdata string
		rlog  string
	}{
		{
			name: "success",
			req: &api.DoesPDVExistRequest{
				Address: "hash",
			},
			f: func(_ context.Context, address string) (bool, error) {
				return true, nil
			},
			rcode: http.StatusOK,
			rdata: `{"exists":true}`,
			rlog:  "",
		},
		{
			name:  "invalid request",
			req:   &api.DoesPDVExistRequest{},
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"request is invalid"}`,
			rlog:  "",
		},
		{
			name: "invalid json",
			req: &api.DoesPDVExistRequest{
				Address:     invalidJSON,
				AuthRequest: api.AuthRequest{Signature: wrongSignature},
			},
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"failed to decode json"}`,
			rlog:  "",
		},
		{
			name: "internal error",
			req: &api.DoesPDVExistRequest{
				Address: "address",
			},
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

			b, w, r := newTestParameters(t)
			pathRequest(t, r, api.DoesPDVExistEndpoint, tc.req, &tc.req.Signature)

			if tc.req.Address == invalidJSON {
				r.Body = ioutil.NopCloser(bytes.NewBufferString(invalidJSON))
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := service.NewMockService(ctrl)

			if tc.f != nil {
				srv.EXPECT().DoesPDVExist(gomock.Any(), tc.req.Address).DoAndReturn(tc.f)
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
			router.Post(api.DoesPDVExistEndpoint, s.doesPDVExistHandler)

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
		req   *api.ReceivePDVRequest
		f     func(_ context.Context, address string) ([]byte, error)
		rcode int
		rdata string
		rlog  string
	}{
		{
			name: "success",
			req: &api.ReceivePDVRequest{
				Address: "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/hash",
			},
			f: func(_ context.Context, address string) ([]byte, error) {
				return []byte("data"), nil
			},
			rcode: http.StatusOK,
			rdata: `{"data":"ZGF0YQ=="}`,
			rlog:  "",
		},
		{
			name:  "invalid request",
			req:   &api.ReceivePDVRequest{},
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"request is invalid"}`,
			rlog:  "",
		},
		{
			name: "invalid json",
			req: &api.ReceivePDVRequest{
				Address:     invalidJSON,
				AuthRequest: api.AuthRequest{Signature: wrongSignature},
			},
			f:     nil,
			rcode: http.StatusBadRequest,
			rdata: `{"error":"failed to decode json"}`,
			rlog:  "",
		},
		{
			name: "not found",
			req: &api.ReceivePDVRequest{
				Address: "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/address",
			},
			f: func(_ context.Context, address string) ([]byte, error) {
				return nil, service.ErrNotFound
			},
			rcode: http.StatusNotFound,
			rdata: `{"error":"PDV 'eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/address' not found"}`,
			rlog:  "",
		},
		{
			name: "internal error",
			req: &api.ReceivePDVRequest{
				Address: "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/address",
			},
			f: func(_ context.Context, address string) ([]byte, error) {
				return nil, errors.New("test error")
			},
			rcode: http.StatusInternalServerError,
			rdata: `{"error":"internal error"}`,
			rlog:  "test error",
		},
		{
			name: "forbidden error",
			req: &api.ReceivePDVRequest{
				Address: "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed7/address",
			},
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

			b, w, r := newTestParameters(t)
			pathRequest(t, r, api.ReceivePDVEndpoint, tc.req, &tc.req.Signature)

			if tc.req.Address == invalidJSON {
				r.Body = ioutil.NopCloser(bytes.NewBufferString(invalidJSON))
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := service.NewMockService(ctrl)

			if tc.f != nil {
				srv.EXPECT().ReceivePDV(gomock.Any(), tc.req.Address).DoAndReturn(tc.f)
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
			router.Post(api.ReceivePDVEndpoint, s.receivePDVHandler)

			router.ServeHTTP(w, r)

			assert.True(t, strings.Contains(b.String(), tc.rlog))
			assert.Equal(t, tc.rcode, w.Code)
			assert.Equal(t, tc.rdata, w.Body.String())
		})
	}
}
