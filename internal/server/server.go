// Package server Cerberus
//
// The Cerberus is an users' data keeper. The Cerberus encrypts data and pushes it into [ipfs](https://ipfs.io)
//
//     Schemes: https
//     BasePath: /v1
//     Version: 1.0.0
//
//     Produces:
//     - application/json
//     Consumes:
//     - application/json
//
//     Security:
//     - public_key:
//     - signature:
//
//     SecurityDefinitions:
//     public_key:
//          type: apiKey
//          name: Public-Key
//          in: header
//          description: Blockchain account's public key
//     signature:
//          type: apiKey
//          name: Signature
//          in: header
//          description: |-
//            Signature of request digest.
//            Digest is sha256 sum of request: {body as is}+{request uri}.
//            For example:
//               Private key in hex: 0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20
//               Request url: localhost/v1/pdv/123-321
//               Body: {"some":"file"}
//               Digest will be made from `{"some":"file"}v1/pdv/123-321`
//               Digest in hex: 772d91d6dd8ff9c93cbd3bc888825f932964b301b32581c1e1dc36cdb5a5d869
//
// swagger:meta
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
)

//go:generate swagger generate spec -t swagger -m -c . -o ../../static/swagger.json

const existenceCacheSize = 200 // we don't need store a lot keys because method used by blockchain node which validates block

var cdc = amino.NewCodec() // nolint:gochecknoglobals

func init() { // nolint:gochecknoinits
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)
}

type server struct {
	s service.Service

	pdvExistenceCache *lru.ARCCache
}

// SetupRouter setups handlers to chi router.
func SetupRouter(s service.Service, r chi.Router, maxBodySize int64) {
	r.Use(
		swaggerMiddleware,
		loggerMiddleware,
		setHeadersMiddleware,
		middleware.StripSlashes,
		recovererMiddleware,
		bodyLimiterMiddleware(maxBodySize),
	)

	c, err := lru.NewARC(existenceCacheSize)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create cache")
	}

	srv := server{
		s:                 s,
		pdvExistenceCache: c,
	}

	r.Post("/v1/pdv", srv.savePDVHandler)
	r.Get("/v1/pdv/{address}", srv.receivePDVHandler)
	r.Head("/v1/pdv/{address}", srv.doesPDVExistHandler)
}

func getLogger(ctx context.Context) logrus.FieldLogger {
	return ctx.Value(logCtxKey{}).(logrus.FieldLogger)
}

func writeErrorf(w http.ResponseWriter, status int, format string, args ...interface{}) {
	body, _ := json.Marshal(api.Error{
		Error: fmt.Sprintf(format, args...),
	})

	w.WriteHeader(status)
	// nolint:gosec,errcheck
	w.Write(body)
}

func writeError(w http.ResponseWriter, s int, message string) {
	writeErrorf(w, s, message)
}

func writeInternalError(l logrus.FieldLogger, w http.ResponseWriter, message string) {
	l.Error(string(debug.Stack()))
	l.Error(message)
	// We don't want to expose internal error to user. So we will just send typical error.
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeOK(w http.ResponseWriter, status int, v interface{}) {
	body, _ := json.Marshal(v)

	w.WriteHeader(status)
	// nolint:gosec,errcheck
	w.Write(body)
}

func writeVerifyError(l logrus.FieldLogger, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, api.ErrNotVerified):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, api.ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeInternalError(l, w, err.Error())
	}
}
