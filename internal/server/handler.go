package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
)

// sendPDVHandler encrypts and puts PDV data into storage.
func (s *server) sendPDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /send-pdv Cerberus SendPDV
	//
	// Encrypts and puts PDV data into storage.
	//
	// ---
	// parameters:
	// - name: request
	//   in: body
	//   required: true
	//   schema:
	//     '$ref': '#/definitions/sendPDVRequest'
	// responses:
	//   '201':
	//     description: pdv was put into storage
	//     schema:
	//       "$ref": "#/definitions/sendPDVResponse"
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
	var req api.SendPDVRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "failed to decode json")
		return
	}

	if !req.IsValid() {
		writeError(w, http.StatusBadRequest, "request is invalid")
		return
	}

	digest, err := api.Verify(req)
	if err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	filepath := fmt.Sprintf("%s/%s", req.Signature.PublicKey, hex.EncodeToString(digest))
	if err := s.s.SendPDV(r.Context(), req.Data, filepath); err != nil {
		writeInternalError(getLogger(r.Context()), w, err.Error())
		return
	}

	writeOK(w, http.StatusCreated, api.SendPDVResponse{Address: filepath})
}

// receivePDVHandler gets pdv from storage and decrypts it.
func (s *server) receivePDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /receive-pdv Cerberus ReceivePDV
	//
	// Gets and decrypts pdv from storage.
	//
	// ---
	// parameters:
	// - name: request
	//   in: body
	//   required: true
	//   schema:
	//     '$ref': '#/definitions/receivePDVRequest'
	// responses:
	//   '200':
	//     description: pdv from storage
	//     schema:
	//       "$ref": "#/definitions/receivePDVResponse"
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
	var req api.ReceivePDVRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "failed to decode json")
		return
	}

	if !req.IsValid() {
		writeError(w, http.StatusBadRequest, "request is invalid")
		return
	}

	if _, err := api.Verify(req); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	if pk := strings.Split(req.Address, "/")[0]; pk != req.Signature.PublicKey {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	data, err := s.s.ReceivePDV(r.Context(), req.Address)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeErrorf(w, http.StatusNotFound, fmt.Sprintf("PDV '%s' not found", req.Address))
		} else {
			writeInternalError(getLogger(r.Context()), w, err.Error())
		}
		return
	}

	writeOK(w, http.StatusOK, api.ReceivePDVResponse{Data: data})
}

// doesPDVExistHandler checks if pdv exists in storage.
func (s *server) doesPDVExistHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pdv-exists Cerberus DoesPDVExist
	//
	// Checks if pdv exists in storage.
	//
	// ---
	// parameters:
	// - name: request
	//   in: body
	//   required: true
	//   schema:
	//     '$ref': '#/definitions/doesPDVExistRequest'
	// responses:
	//   '200':
	//     description: result of check
	//     schema:
	//       "$ref": "#/definitions/doesPDVExistResponse"
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
	var req api.DoesPDVExistRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "failed to decode json")
		return
	}

	if !req.IsValid() {
		writeError(w, http.StatusBadRequest, "request is invalid")
		return
	}

	if _, err := api.Verify(req); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	exists, err := s.s.DoesPDVExist(r.Context(), req.Address)
	if err != nil {
		writeInternalError(getLogger(r.Context()), w, err.Error())
		return
	}

	writeOK(w, http.StatusOK, api.DoesPDVExistResponse{Exists: exists})
}
