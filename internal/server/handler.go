package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
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

	if err := api.Verify(r); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	var p schema.PDV
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("request is invalid: %s", err.Error()))
		return
	}

	if !p.Validate() {
		writeError(w, http.StatusBadRequest, "pdv data is invalid")
		return
	}

	if l := uint16(len(p.PDV)); l < s.minPDVCount || l > s.maxPDVCount {
		writeError(w, http.StatusBadRequest, "forbidden pdv count")
		return
	}

	owner, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
	if err != nil {
		writeInternalError(getLogger(r.Context()), w, fmt.Sprintf("failed to decode owner address: %s", err.Error()))
		return
	}

	id, meta, err := s.s.SavePDV(r.Context(), p, owner)
	if err != nil {
		writeInternalError(getLogger(r.Context()).WithError(err), w, "failed to save pdv")
		return
	}

	s.pdvMetaCache.Add(getCacheKey(owner.String(), id), meta)

	writeOK(w, http.StatusCreated, api.SavePDVResponse{ID: id})
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
		writeError(w, http.StatusBadRequest, "invalid owner")
		return
	}

	var err error

	var from uint64
	if s := r.URL.Query().Get("from"); s != "" {
		if from, err = strconv.ParseUint(s, 10, 64); err != nil {
			writeError(w, http.StatusBadRequest, "invalid from")
			return
		}
	}

	limit := defaultLimit
	if s := r.URL.Query().Get("limit"); s != "" {
		if limit, err = strconv.ParseUint(s, 10, 16); err != nil || limit > 1000 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
	}

	list, err := s.s.ListPDV(r.Context(), owner, from, uint16(limit))
	if err != nil {
		writeInternalError(getLogger(r.Context()).WithError(err), w, "failed to list pdv")
		return
	}

	data, err := json.Marshal(list)
	if err != nil {
		writeInternalError(getLogger(r.Context()).WithError(err), w, "failed to marshal list of pdv")
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
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if !isOwnerValid(chi.URLParam(r, "owner")) {
		writeError(w, http.StatusBadRequest, "invalid owner")
		return
	}

	if err := api.Verify(r); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	owner, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to generate address")
		return
	}

	if chi.URLParam(r, "owner") != owner.String() {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	data, err := s.s.ReceivePDV(r.Context(), owner.String(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeErrorf(w, http.StatusNotFound, fmt.Sprintf("PDV '%d' not found", id))
		} else {
			writeInternalError(getLogger(r.Context()), w, err.Error())
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
		writeError(w, http.StatusBadRequest, "invalid address")
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var m api.PDVMeta
	if v, ok := s.pdvMetaCache.Get(getCacheKey(owner, id)); ok {
		m = v.(api.PDVMeta) // nolint
	} else {
		var err error
		m, err = s.s.GetPDVMeta(r.Context(), owner, id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeErrorf(w, http.StatusNotFound, fmt.Sprintf("PDV '%d' not found", id))
				return
			}
			writeInternalError(getLogger(r.Context()), w, err.Error())
			return
		}
		s.pdvMetaCache.Add(getCacheKey(owner, id), m)
	}

	writeOK(w, http.StatusOK, m)
}

func getAddressFromPubKey(k string) (sdk.AccAddress, error) {
	var pk secp256k1.PubKeySecp256k1
	b, err := hex.DecodeString(k)
	if err != nil {
		return nil, err
	}

	if err := cdc.UnmarshalBinaryBare(api.GetAminoSecp256k1PubKey(b), &pk); err != nil {
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
