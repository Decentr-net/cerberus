package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"github.com/tomasen/realip"
)

type logCtxKey struct{}

// recovererMiddleware handles panics.
func recovererMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				log := getLogger(r.Context())
				log.Error("service recovered from panic")
				log.Error("panic:")
				log.Error(spew.Sdump(rvr))

				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// loggerMiddleware puts logger with client's info into context.
func loggerMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log := logrus.WithField("ip", realip.FromRequest(r))

		ctx := context.WithValue(r.Context(), logCtxKey{}, log)
		next.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}

// setHeadersMiddleware sets predefined headers to response.
func setHeadersMiddleware(handler http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler.ServeHTTP(w, r)
	})

	return http.Handler(fn)
}

// swaggerMiddleware for swagger-ui.
func swaggerMiddleware(next http.Handler) http.Handler {
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
		next.ServeHTTP(w, r)
	})
}

// bodyLimiterMiddleware returns middleware which limits size of data read from request's body.
func bodyLimiterMiddleware(maxBodySize int64) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

			next.ServeHTTP(w, r)
		})

		return fn
	}
}
