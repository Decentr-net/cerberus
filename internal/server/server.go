// Package server Cerberus
//
// The Cerberus is an users' data keeper. The Cerberus encrypts data and pushes it into S3.
//
//     Schemes: https
//     BasePath: /v1
//     Version: 1.0.1
//
//     Produces:
//     - application/json
//     Consumes:
//     - application/json
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
//            Signature of request digest.<br>
//            Digest is sha256 sum of request: `{body as is}`+`{request uri}`.<br>
//            For example:<br>
//            Private key in hex: ```cfe43c70347c7e39084612d9448f3ed86ed733a33a67de35c7e335b3c4edc37d```<br>
//            Request url: ```http://localhost/v1/pdv```<br>
//            Body: ```{"some":"file"}```<br>
//            Digest will be made from ```{"some":"file"}/v1/pdv```<br>
//            Digest in hex:<br>
//            ```4a1084d05820d60aee9ce600227ca2290ef63e80e5227215b58b023ec6876799```<br>
//            Signature in hex:<br>
//            ```28eff4676d7839648dda925ba92d447dd7552e177a302f32681fc76278088f9f1fb98051666aa02dd80f7d9b7c01d42ea1abbb3e65de8f1fd04be7b747fb0692```<br>
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

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
	_ "github.com/Decentr-net/cerberus/pkg/api/swagger" // import models to be generated into swagger.json
)

//go:generate swagger generate spec -t swagger -m -c . -o ../../static/swagger.json

// nolint: gochecknoinits
func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("decentr", "decentrpub") // from decentr.app package
	config.Seal()
}

const existenceCacheSize = 100000

const defaultLimit uint64 = 100

var cdc = amino.NewCodec() // nolint:gochecknoglobals

func init() { // nolint:gochecknoinits
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)
}

type server struct {
	s service.Service

	pdvMetaCache *lru.ARCCache

	minPDVCount uint16
	maxPDVCount uint16
}

// SetupRouter setups handlers to chi router.
func SetupRouter(s service.Service, r chi.Router, maxBodySize int64, minPDVCount, maxPDVCount uint16) {
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
		s:            s,
		pdvMetaCache: c,

		minPDVCount: minPDVCount,
		maxPDVCount: maxPDVCount,
	}

	r.Post("/v1/pdv", srv.savePDVHandler)
	r.Get("/v1/pdv/{owner}", srv.listPDVHandler)
	r.Get("/v1/pdv/{owner}/{id}", srv.getPDVHandler)
	r.Get("/v1/pdv/{owner}/{id}/meta", srv.getPDVMetaHandler)
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

func isOwnerValid(s string) bool {
	_, err := sdk.AccAddressFromBech32(s)
	return err == nil
}
