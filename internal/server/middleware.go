package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/tomasen/realip"

	logging "github.com/Decentr-net/logrus/context"
)

// recovererMiddleware handles panics.
func recovererMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				logging.GetLogger(r.Context()).Info("service recovered from panic")

				writeInternalError(r.Context(), w, spew.Sdump(rvr))
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

		next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), log)))
	}

	return http.HandlerFunc(fn)
}

// requestIDMiddleware puts request-id to headers and adds it into a logger.
func requestIDMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		id := uuid.Must(uuid.NewV4()).String()

		w.Header().Set("X-Request-ID", id)
		l := logging.GetLogger(r.Context()).WithField("request_id", id)

		next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), l)))
	}

	return http.HandlerFunc(fn)
}

// timeoutMiddleware puts timeout context into request
func timeoutMiddleware(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logging.GetLogger(r.Context()).WithField("url", r.URL.String()).Debug("start processing")

			ctx, _ := context.WithTimeout(r.Context(), timeout)
			next.ServeHTTP(w, r.WithContext(ctx))

			logging.GetLogger(r.Context()).WithField("elapsed_time", time.Since(start)).Debug("processed")
		}

		return http.HandlerFunc(fn)
	}
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
