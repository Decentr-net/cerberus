package server

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"github.com/tomasen/realip"
)

type logCtxKey struct{}

// Recoverer middleware handles panics.
func Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); r != nil {
				log := getLogger(r.Context())
				log.Error("service recovered from panic")
				log.Error("stacktrace:")
				log.Error(string(debug.Stack()))
				log.Error("panic:")
				log.Error(spew.Sdump(rvr))

				writeInternalError(log, w, "panic: internal error")
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// Logger middleware puts logger with client's info into context.
func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log := logrus.WithField("ip", realip.FromRequest(r))

		ctx := context.WithValue(r.Context(), logCtxKey{}, log)
		next.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

// SetHeaders middleware sets predefined headers to response.
func SetHeaders(handler http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		handler.ServeHTTP(w, r)
	})

	return http.Handler(fn)
}

// Swagger middleware for swagger-ui.
func Swagger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Shortcut helpers for swagger-ui
		if r.URL.Path == "/docs" {
			http.Redirect(w, r, "/docs/", http.StatusFound)
			return
		}
		// Serving ./swagger-ui/
		if strings.Index(r.URL.Path, "/docs/") == 0 {
			http.StripPrefix("/docs/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
