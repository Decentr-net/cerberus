// Package server Cerberus
//
// The Cerberus is an users' data keeper. The Cerberus encrypts data and pushes it into S3.
//
//     Schemes: https
//     BasePath: /v1
//     Version: 1.2.1
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
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/Decentr-net/go-api"

	_ "github.com/Decentr-net/cerberus/internal/server/swagger" // import models to be generated into swagger.json
	"github.com/Decentr-net/cerberus/internal/service"
)

//go:generate swagger generate spec -t swagger -m -c . -o ../../static/swagger.json

const (
	existenceCacheSize        = 100000
	defaultLimit       uint64 = 100

	dateFormat = "2006-01-02"
)

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

// Profile ...
// swagger:model APIProfile
type Profile struct {
	Address   string   `json:"address"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Emails    []string `json:"emails,omitempty"`
	Bio       string   `json:"bio"`
	Gender    string   `json:"gender"`
	Avatar    string   `json:"avatar"`
	Birthday  string   `json:"birthday"`
	CreatedAt int64    `json:"createdAt"`
}

// SetupRouter setups handlers to chi router.
func SetupRouter(s service.Service, r chi.Router, timeout time.Duration, maxBodySize int64, minPDVCount, maxPDVCount uint16) {
	r.Use(
		api.FileServerMiddleware("/docs", "static"),
		api.LoggerMiddleware,
		middleware.StripSlashes,
		cors.AllowAll().Handler,
		api.RequestIDMiddleware,
		api.RecovererMiddleware,
		api.TimeoutMiddleware(timeout),
		api.BodyLimiterMiddleware(maxBodySize),
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
	r.Get("/v1/profiles", srv.getProfilesHandler)

	r.Get("/v1/configs/rewards", srv.getRewardsConfigHandler)
}

func isOwnerValid(s string) bool {
	_, err := sdk.AccAddressFromBech32(s)
	return err == nil
}
