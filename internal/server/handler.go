package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chi/chi"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

// sendPDVHandler encrypts and puts PDV data into storage.
func (s *server) sendPDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pdv Cerberus SendPDV
	//
	// Encrypts and puts PDV data into storage.
	//
	// ---
	// security:
	// - public_key: []
	// - signature: []
	// produces:
	// - application/json
	// consumes:
	// - application/octet-stream
	// parameters:
	// - name: request
	//   in: body
	//   required: true
	//   schema:
	//     type: file
	// responses:
	//   '201':
	//     description: pdv was put into storage
	//     schema:
	//       "$ref": "#/definitions/SendPDVResponse"
	//   '401':
	//     description: signature wasn't verified
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

	digest, err := api.Verify(r)
	if err != nil {
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

	if !p.PDV.Validate() {
		writeError(w, http.StatusBadRequest, "pdv data is invalid")
		return
	}

	filepath := fmt.Sprintf("%s-%s", r.Header.Get(api.PublicKeyHeader), hex.EncodeToString(digest))
	if err := s.s.SendPDV(r.Context(), data, filepath); err != nil {
		writeInternalError(getLogger(r.Context()), w, err.Error())
		return
	}

	s.pdvExistenceCache.Add(filepath, true)

	writeOK(w, http.StatusCreated, api.SendPDVResponse{Address: filepath})
}

// receivePDVHandler gets pdv from storage and decrypts it.
func (s *server) receivePDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /pdv/{address} Cerberus ReceivePDV
	//
	// Gets and decrypts PDV from storage.
	//
	// ---
	// produces:
	// - application/octet-stream
	// - application/json
	// security:
	// - public_key: []
	// - signature: []
	// parameters:
	// - name: address
	//   description: PDV's address
	//   in: path
	//   required: true
	//   type: string
	// responses:
	//   '200':
	//     description: PDV from storage
	//     schema:
	//       type: file
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

	if _, err := api.Verify(r); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	if pk := strings.Split(address, "-")[0]; pk != r.Header.Get(api.PublicKeyHeader) {
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

// doesPDVExistHandler checks if pdv exists in storage.
func (s *server) doesPDVExistHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation HEAD /pdv/{address} Cerberus DoesPDVExist
	//
	// Checks if PDV exists in storage.
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
	//     description: PDV exists
	//   '404':
	//     description: PDV doesn't exist
	//   '401':
	//     description: signature wasn't verified
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

	var exists bool
	if v, ok := s.pdvExistenceCache.Get(address); ok {
		exists = v.(bool) // nolint
	} else {
		var err error
		exists, err = s.s.DoesPDVExist(r.Context(), address)
		if err != nil {
			writeInternalError(getLogger(r.Context()), w, err.Error())
			return
		}
		s.pdvExistenceCache.Add(address, exists)
	}

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
