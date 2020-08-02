package server

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_recovererMiddleware(t *testing.T) {
	b, w, r := newTestParameters(t, http.MethodGet, "", nil)

	recovererMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		panic("some panic")
	})).ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"error":"internal error"}`, w.Body.String())
	assert.True(t, strings.Contains(b.String(), "some panic"))
}

func Test_loggerMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodPost, "", nil)
	require.NoError(t, err)

	loggerMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, ir *http.Request) {
		l := getLogger(ir.Context())
		assert.NotNil(t, l)
	})).ServeHTTP(w, r)
}

func Test_setHeadersMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodPost, "", nil)
	require.NoError(t, err)

	setHeadersMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, ir *http.Request) {})).ServeHTTP(w, r)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func Test_bodyLimiterMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(make([]byte, 10000)))
	require.NoError(t, err)

	bodyLimiterMiddleware(1000)(http.HandlerFunc(func(_ http.ResponseWriter, ir *http.Request) {
		_, err := ioutil.ReadAll(ir.Body)
		assert.Error(t, err)
		assert.Equal(t, "http: request body too large", err.Error())
	})).ServeHTTP(w, r)
}
