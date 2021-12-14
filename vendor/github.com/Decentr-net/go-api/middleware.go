package api

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/tomasen/realip"

	logging "github.com/Decentr-net/logrus/context"
)

// RecovererMiddleware handles panics.
func RecovererMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				logging.GetLogger(r.Context()).Infof(
					"service recovered from panic stack=%s", string(debug.Stack()))

				WriteInternalError(r.Context(), w, spew.Sdump(rvr))
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// LoggerMiddleware puts logger with client's info into context.
func LoggerMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log := logrus.WithField("ip", realip.FromRequest(r))

		next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), log)))
	}

	return http.HandlerFunc(fn)
}

// RequestIDMiddleware puts request-id to headers and adds it into a logger.
func RequestIDMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		id := uuid.Must(uuid.NewV4()).String()

		w.Header().Set("X-Request-ID", id)
		l := logging.GetLogger(r.Context()).WithField("request_id", id)

		next.ServeHTTP(w, r.WithContext(logging.WithLogger(r.Context(), l)))
	}

	return http.HandlerFunc(fn)
}

// TimeoutMiddleware puts timeout context into request.
func TimeoutMiddleware(timeout time.Duration) func(next http.Handler) http.Handler {
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

// FileServerMiddleware serves requests with prefix into directory.
func FileServerMiddleware(prefix, dir string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Shortcut helpers for swagger-ui
			if r.URL.Path == prefix {
				http.Redirect(w, r, fmt.Sprintf("%s/", prefix), http.StatusFound)
				return
			}
			// Serving ./swagger-ui/
			if strings.Index(r.URL.Path, fmt.Sprintf("%s/", prefix)) == 0 {
				http.StripPrefix(fmt.Sprintf("%s/", prefix), http.FileServer(http.Dir(dir))).ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimiterMiddleware returns middleware which limits size of data read from request's body.
func BodyLimiterMiddleware(maxBodySize int64) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

			next.ServeHTTP(w, r)
		})

		return fn
	}
}
