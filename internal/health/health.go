// Package health provides handler for health checks.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/Decentr-net/go-api"
)

// nolint:gochecknoglobals
var (
	version = "dev"
	commit  = "unknown"
)

// GetVersion returns service's version and commit.
func GetVersion() string {
	return fmt.Sprintf("%s-%s", version, commit)
}

// VersionResponse ...
type VersionResponse struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// Pinger pings external service.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingFunc is wrapper for raw func.
type PingFunc func(ctx context.Context) error

// Ping ...
func (f PingFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

// SetupRouter setups all pingers to /health.
func SetupRouter(r chi.Router, p ...Pinger) {
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := context.WithTimeout(r.Context(), time.Second*5) // nolint:govet
		gr, ctx := errgroup.WithContext(ctx)

		for i := range p {
			v := p[i]
			gr.Go(func() error {
				if err := v.Ping(ctx); err != nil {
					logrus.WithError(err).Error("health check failed")
					return err
				}
				return nil
			})
		}

		if err := gr.Wait(); err != nil {
			data, _ := json.Marshal(struct {
				api.Error
				VersionResponse
			}{
				Error:           api.Error{Error: err.Error()},
				VersionResponse: VersionResponse{Version: version, Commit: commit},
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(data) // nolint

			return
		}
		data, _ := json.Marshal(VersionResponse{Version: version, Commit: commit})

		w.WriteHeader(http.StatusOK)
		w.Write(data) // nolint
	})
}
