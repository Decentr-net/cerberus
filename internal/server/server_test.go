package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/Decentr-net/cerberus/pkg/api"
)

func Test_getLogger(t *testing.T) {
	ctx := context.WithValue(context.Background(), logCtxKey{}, logrus.WithError(nil))
	require.NotNil(t, getLogger(ctx))
}

func Test_writeOK(t *testing.T) {
	w := httptest.NewRecorder()
	writeOK(w, http.StatusCreated, struct {
		M int
		N string
	}{
		M: 5,
		N: "str",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, `{"M":5,"N":"str"}`, w.Body.String())
}

func Test_writeError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusNotFound, "some error")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, `{"error":"some error"}`, w.Body.String())
}

func Test_writeErrorf(t *testing.T) {
	w := httptest.NewRecorder()
	writeErrorf(w, http.StatusForbidden, "some error %d", 1)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, `{"error":"some error 1"}`, w.Body.String())
}

func Test_writeInternalError(t *testing.T) {
	b, w, r := newTestParameters(t, http.MethodGet, "", nil)

	writeInternalError(getLogger(r.Context()), w, "some error")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Greater(t, len(b.String()), 20) // stacktrace
	assert.True(t, strings.Contains(b.String(), "some error"))
	assert.Equal(t, `{"error":"internal error"}`, w.Body.String())
}

func Test_writeVerifyError(t *testing.T) {
	t.Run("bad request", func(t *testing.T) {
		b, w, r := newTestParameters(t, http.MethodGet, "", nil)

		writeVerifyError(getLogger(r.Context()), w, api.ErrInvalidPublicKey)

		assert.Empty(t, b.String())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, `{"error":"invalid request: public key is invalid"}`, w.Body.String())
	})

	t.Run("not verified", func(t *testing.T) {
		b, w, r := newTestParameters(t, http.MethodGet, "", nil)
		writeVerifyError(getLogger(r.Context()), w, api.ErrNotVerified)

		assert.Empty(t, b.String())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"error":"failed to verify message"}`, w.Body.String())
	})

	t.Run("internal error", func(t *testing.T) {
		b, w, r := newTestParameters(t, http.MethodGet, "", nil)

		writeVerifyError(getLogger(r.Context()), w, errors.New("some error"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Greater(t, len(b.String()), 20) // stacktrace
		assert.True(t, strings.Contains(b.String(), "some error"))
		assert.Equal(t, `{"error":"internal error"}`, w.Body.String())
	})
}
