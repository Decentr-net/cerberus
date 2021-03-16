package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	logging "github.com/Decentr-net/logrus/context"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_recovererMiddleware(t *testing.T) {
	b, w, r := newTestParameters(t, http.MethodGet, "", nil)

	require.NotPanics(t, func() {
		recovererMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			panic("some panic")
		})).ServeHTTP(w, r)
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"error":"internal error"}`, w.Body.String())
	assert.Contains(t, b.String(), "some panic")
}

func Test_loggerMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodPost, "", nil)
	require.NoError(t, err)

	loggerMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, ir *http.Request) {
		l := logging.GetLogger(ir.Context())
		assert.NotNil(t, l)
	})).ServeHTTP(w, r)
}

func Test_setHeadersMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	m := chi.NewMux()
	m.Use(setHeadersMiddleware)
	m.Get("/", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(`{"json": "json"}`))
	})
	m.ServeHTTP(w, r)

	assert.Equal(t, "application/json", w.Result().Header.Get("Content-Type")) // nolint
}

func Test_requestIDMiddleware(t *testing.T) {
	l := logrus.New()
	b := bytes.NewBufferString("")
	l.SetOutput(b)

	w := httptest.NewRecorder()
	r, err := http.NewRequestWithContext(logging.WithLogger(context.Background(), l), http.MethodGet, "/", nil)
	require.NoError(t, err)

	var id string

	requestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		id = w.Header().Get("X-Request-ID")

		logging.GetLogger(r.Context()).Info("hi")
	})).ServeHTTP(w, r)

	assert.NotEmpty(t, id)
	assert.Contains(t, b.String(), fmt.Sprintf("request_id=%s", id))
}

func Test_timeoutMiddleware(t *testing.T) {
	l := logrus.New()
	b := bytes.NewBufferString("")
	l.SetLevel(logrus.DebugLevel)
	l.SetOutput(b)

	w := httptest.NewRecorder()
	r, err := http.NewRequestWithContext(logging.WithLogger(context.Background(), l), http.MethodGet, "/", nil)
	require.NoError(t, err)

	timeoutMiddleware(time.Millisecond*5)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		select {
		case <-r.Context().Done():
			require.True(t, errors.Is(r.Context().Err(), context.DeadlineExceeded))
		case <-ctx.Done():
			assert.Fail(t, "should be timed out")
		}
	})).ServeHTTP(w, r)

	s := regexp.MustCompile(`elapsed_time="?(.+)"?`).FindStringSubmatch(b.String())
	require.Len(t, s, 2)

	tt, err := time.ParseDuration(s[1])
	require.NoError(t, err)
	require.NotZero(t, tt.Milliseconds())
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
