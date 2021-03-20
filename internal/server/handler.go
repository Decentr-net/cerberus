package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/Decentr-net/go-api"
	logging "github.com/Decentr-net/logrus/context"

	"github.com/Decentr-net/cerberus/internal/service"
	capi "github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

// savePDVHandler encrypts and puts PDV data into storage.
func (s *server) savePDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pdv PDV Save
	//
	// Encrypts and saves PDV
	//
	// ---
	// security:
	// - public_key: []
	//   signature: []
	// produces:
	// - application/json
	// consumes:
	// - application/json
	// parameters:
	// - name: request
	//   description: batch of pdv
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/PDV"
	// responses:
	//   '201':
	//     description: pdv was put into storage
	//     schema:
	//       "$ref": "#/definitions/SavePDVResponse"
	//   '401':
	//     description: signature wasn't verified
	//     schema:
	//       "$ref": "#/definitions/Error"
	//   '400':
	//      description: bad request
	//      schema:
	//        "$ref": "#/definitions/Error"
	//   '500':
	//      description: internal server error
	//      schema:
	//        "$ref": "#/definitions/Error"

	if err := capi.Verify(r); err != nil {
		api.WriteVerifyError(r.Context(), w, err)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to read body: %s", err.Error()))
		return
	}
	r.Body.Close() // nolint:errcheck,gosec

	var p schema.PDV
	if err := json.Unmarshal(data, &p); err != nil {
		logging.GetLogger(r.Context()).WithField("body", string(data)).Debug("failed to decode pdv")
		api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("request is invalid: %s", err.Error()))
		return
	}

	if !p.Validate() {
		logging.GetLogger(r.Context()).WithField("body", string(data)).Debug("failed to validate pdv")
		api.WriteError(w, http.StatusBadRequest, "pdv data is invalid")
		return
	}

	if l := uint16(len(p.PDV)); l < s.minPDVCount || l > s.maxPDVCount {
		api.WriteError(w, http.StatusBadRequest, "forbidden pdv count")
		return
	}

	owner, err := getAddressFromPubKey(r.Header.Get(capi.PublicKeyHeader))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to decode owner address: %s", err.Error()))
		return
	}

	id, meta, err := s.s.SavePDV(r.Context(), p, owner)
	if err != nil {
		api.WriteInternalErrorf(r.Context(), w, "failed to save pdv: %s", err.Error())
		return
	}

	s.pdvMetaCache.Add(getCacheKey(owner.String(), id), meta)

	api.WriteOK(w, http.StatusCreated, capi.SavePDVResponse{ID: id})
}

// listPDVHandler lists pdv from storage.
func (s *server) listPDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /pdv/{owner} PDV List
	//
	// Lists PDV
	//
	// ---
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   description: PDV's address
	//   in: path
	//   required: true
	//   example: decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz
	//   type: string
	// - name: from
	//   description: id of PDV to start from
	//   in: query
	//   type: integer
	//   format: uint64
	// - name: limit
	//   description: how many pdv will be returned
	//   in: query
	//   type: integer
	//   format: uint16
	//   maximum: 1000
	// responses:
	//   '200':
	//     description: List of PDV
	//     schema:
	//       type: array
	//       items:
	//         type: integer
	//         format: uint64
	//   '400':
	//     description: bad request
	//     schema:
	//       "$ref": "#/definitions/Error"
	//   '500':
	//     description: internal server error
	//     schema:
	//       "$ref": "#/definitions/Error"

	owner := chi.URLParam(r, "owner")
	if !isOwnerValid(owner) {
		api.WriteError(w, http.StatusBadRequest, "invalid owner")
		return
	}

	var err error

	var from uint64
	if s := r.URL.Query().Get("from"); s != "" {
		if from, err = strconv.ParseUint(s, 10, 64); err != nil {
			api.WriteError(w, http.StatusBadRequest, "invalid from")
			return
		}
	}

	limit := defaultLimit
	if s := r.URL.Query().Get("limit"); s != "" {
		if limit, err = strconv.ParseUint(s, 10, 16); err != nil || limit > 1000 {
			api.WriteError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}

	list, err := s.s.ListPDV(r.Context(), owner, from, uint16(limit))
	if err != nil {
		api.WriteInternalErrorf(r.Context(), w, "failed to list pdv: %s", err.Error())
		return
	}

	data, err := json.Marshal(list)
	if err != nil {
		api.WriteInternalErrorf(r.Context(), w, "failed to marshal list of pdv: %s", err.Error())
		return
	}

	w.Write(data) // nolint
}

// getPDVHandler gets pdv from storage and decrypts it.
func (s *server) getPDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /pdv/{owner}/{id} PDV Get
	//
	// Returns plain PDV
	//
	// ---
	// produces:
	// - application/json
	// security:
	// - public_key: []
	//   signature: []
	// parameters:
	// - name: address
	//   description: PDV's address
	//   in: path
	//   required: true
	//   type: string
	// responses:
	//   '200':
	//     description: PDV
	//     schema:
	//       "$ref": "#/definitions/PDV"
	//   '401':
	//     description: signature wasn't verified
	//     schema:
	//       "$ref": "#/definitions/Error"
	//   '403':
	//     description: access to file is denied
	//     schema:
	//       "$ref": "#/definitions/Error"
	//   '400':
	//     description: bad request
	//     schema:
	//       "$ref": "#/definitions/Error"
	//   '500':
	//     description: internal server error
	//     schema:
	//       "$ref": "#/definitions/Error"

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if !isOwnerValid(chi.URLParam(r, "owner")) {
		api.WriteError(w, http.StatusBadRequest, "invalid owner")
		return
	}

	if err := capi.Verify(r); err != nil {
		api.WriteVerifyError(r.Context(), w, err)
		return
	}

	owner, err := getAddressFromPubKey(r.Header.Get(capi.PublicKeyHeader))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "failed to generate address")
		return
	}

	if chi.URLParam(r, "owner") != owner.String() {
		api.WriteError(w, http.StatusForbidden, "access denied")
		return
	}

	data, err := s.s.ReceivePDV(r.Context(), owner.String(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			api.WriteErrorf(w, http.StatusNotFound, fmt.Sprintf("PDV '%d' not found", id))
		} else {
			api.WriteInternalErrorf(r.Context(), w, "failed to receive pdv: %s", err.Error())
		}
		return
	}

	w.Write(data) // nolint
}

// getPDVMetaHandler returns PDVs meta by address.
func (s *server) getPDVMetaHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /pdv/{owner}/{id}/meta PDV GetMeta
	//
	// Get meta
	//
	// Returns metadata of PDV
	//
	// ---
	// parameters:
	// - name: address
	//   description: PDV's address
	//   in: path
	//   required: true
	//   type: string
	// responses:
	//   '200':
	//     description: metadata of pdv
	//     schema:
	//       "$ref": "#/definitions/PDVMeta"
	//   '404':
	//     description: PDV doesn't exist
	//     schema:
	//       "$ref": "#/definitions/Error"
	//   '400':
	//     description: bad request
	//     schema:
	//       "$ref": "#/definitions/Error"
	//   '500':
	//     description: internal server error
	//     schema:
	//       "$ref": "#/definitions/Error"

	owner := chi.URLParam(r, "owner")

	if !isOwnerValid(owner) {
		api.WriteError(w, http.StatusBadRequest, "invalid address")
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var m capi.PDVMeta
	if v, ok := s.pdvMetaCache.Get(getCacheKey(owner, id)); ok {
		logging.GetLogger(r.Context()).WithField("key", getCacheKey(owner, id)).Debug("meta found in cache")
		m = v.(capi.PDVMeta) // nolint
	} else {
		logging.GetLogger(r.Context()).WithField("key", getCacheKey(owner, id)).Debug("meta wasn't found in cache")
		var err error
		m, err = s.s.GetPDVMeta(r.Context(), owner, id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				api.WriteErrorf(w, http.StatusNotFound, fmt.Sprintf("PDV '%d' not found", id))
				return
			}
			api.WriteInternalErrorf(r.Context(), w, "failed to get meta: %s", err.Error())
			return
		}
		s.pdvMetaCache.Add(getCacheKey(owner, id), m)
	}

	api.WriteOK(w, http.StatusOK, m)
}

func getAddressFromPubKey(k string) (sdk.AccAddress, error) {
	var pk secp256k1.PubKeySecp256k1
	b, err := hex.DecodeString(k)
	if err != nil {
		return nil, err
	}

	if err := cdc.UnmarshalBinaryBare(capi.GetAminoSecp256k1PubKey(b), &pk); err != nil {
		return nil, err
	}

	addr, err := sdk.AccAddressFromHex(pk.Address().String())
	if err != nil {
		panic(err)
	}
	return addr, err
}

func getCacheKey(owner string, id uint64) string {
	return fmt.Sprintf("%s-%d", owner, id)
}
