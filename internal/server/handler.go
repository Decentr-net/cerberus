package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

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

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
	}
	r.Body.Close() // nolint

	var p schema.PDV
	if err := json.Unmarshal(data, &p); err != nil {
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

	filepath, err := getPDVFilepath(r.Header.Get(api.PublicKeyHeader), data)
	if err != nil {
		writeInternalError(getLogger(r.Context()), w, fmt.Sprintf("failed to get filepath: %s", err.Error()))
		return
	}

	if err := s.s.SavePDV(r.Context(), p, filepath); err != nil {
		writeInternalError(getLogger(r.Context()), w, err.Error())
		return
	}

	meta, err := s.s.GetPDVMeta(r.Context(), filepath)
	if err != nil {
		writeInternalError(getLogger(r.Context()), w, fmt.Sprintf("failed to get pdv meta: %s", err.Error()))
		return
	}

	s.pdvMetaCache.Add(filepath, meta)

	writeOK(w, http.StatusCreated, api.SavePDVResponse{Address: filepath})
}

// getPDVHandler gets pdv from storage and decrypts it.
func (s *server) getPDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /pdv/{address} PDV Get
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

	address := chi.URLParam(r, "address")
	if !api.IsAddressValid(address) {
		writeError(w, http.StatusBadRequest, "invalid address")
		return
	}

	if err := api.Verify(r); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	ownerAddress, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate address")
	}

	if strings.Split(address, "-")[0] != ownerAddress {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	data, err := s.s.ReceivePDV(r.Context(), address)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeErrorf(w, http.StatusNotFound, fmt.Sprintf("PDV '%s' not found", address))
		} else {
			writeInternalError(getLogger(r.Context()), w, err.Error())
		}
		return
	}

	w.Write(data) // nolint
}

// getPDVMetaHandler returns PDVs meta by address.
func (s *server) getPDVMetaHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /pdv/{address}/meta PDV GetMeta
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

	address := chi.URLParam(r, "address")

	if !api.IsAddressValid(address) {
		writeError(w, http.StatusBadRequest, "invalid address")
		return
	}

	var m api.PDVMeta
	if v, ok := s.pdvMetaCache.Get(address); ok {
		m = v.(api.PDVMeta) // nolint
	} else {
		var err error
		m, err = s.s.GetPDVMeta(r.Context(), address)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeErrorf(w, http.StatusNotFound, fmt.Sprintf("PDV '%s' not found", address))
				return
			}
			writeInternalError(getLogger(r.Context()), w, err.Error())
			return
		}
		s.pdvMetaCache.Add(address, m)
	}

	writeOK(w, http.StatusOK, m)
}

func getAddressFromPubKey(k string) (string, error) {
	var pk secp256k1.PubKeySecp256k1
	b, _ := hex.DecodeString(k)
	if err := cdc.UnmarshalBinaryBare(api.GetAminoSecp256k1PubKey(b), &pk); err != nil {
		return "", err
	}
	return hex.EncodeToString(pk.Address()), nil
}

func getPDVFilepath(pk string, pdv []byte) (string, error) {
	address, err := getAddressFromPubKey(pk)
	if err != nil {
		return "", fmt.Errorf("failed to get address from pubkey: %w", err)
	}

	hash := sha256.Sum256(pdv)
	return fmt.Sprintf("%s-%s", address, hex.EncodeToString(hash[:])), nil
}
