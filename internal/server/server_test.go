package server

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Decentr-net/go-api"
	"github.com/Decentr-net/go-api/test"
)

func Test_writeVerifyError(t *testing.T) {
	t.Run("bad request", func(t *testing.T) {
		b, w, r := test.NewAPITestParameters(http.MethodGet, "", nil)

		api.WriteVerifyError(r.Context(), w, api.ErrInvalidPublicKey)

		assert.Empty(t, b.String())
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.JSONEq(t, `{"error":"invalid request: public key is invalid"}`, w.Body.String())
	})

	t.Run("not verified", func(t *testing.T) {
		b, w, r := test.NewAPITestParameters(http.MethodGet, "", nil)
		api.WriteVerifyError(r.Context(), w, api.ErrNotVerified)

		assert.Empty(t, b.String())
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.JSONEq(t, `{"error":"failed to verify message"}`, w.Body.String())
	})

	t.Run("internal error", func(t *testing.T) {
		b, w, r := test.NewAPITestParameters(http.MethodGet, "", nil)

		api.WriteVerifyError(r.Context(), w, errors.New("some error"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Greater(t, len(b.String()), 20) // stacktrace
		assert.True(t, strings.Contains(b.String(), "some error"))
		assert.JSONEq(t, `{"error":"internal error"}`, w.Body.String())
	})
}
