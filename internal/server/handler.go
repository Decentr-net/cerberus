package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
	//     description: data were put into storage
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

	if err := api.Verify(req); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	address, err := s.s.SendPDV(r.Context(), req.Data)
	if err != nil {
		writeInternalError(getLogger(r.Context()), w, err.Error())
		return
	}

	writeOK(w, http.StatusCreated, api.SendPDVResponse{Address: address})
}

// receivePDVHandler gets PDV data from storage and decrypts it.
func (s *server) receivePDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /receive-pdv Cerberus ReceivePDV
	//
	// Gets and decrypts PDV data from storage.
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
	//     description: raw data from storage
	//     schema:
	//       "$ref": "#/definitions/receivePDVResponse"
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
	var req api.ReceivePDVRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "failed to decode json")
		return
	}

	if !req.IsValid() {
		writeError(w, http.StatusBadRequest, "request is invalid")
		return
	}

	if err := api.Verify(req); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
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

// DoesPDVExistHandler checks if PDV data exists in storage.
func (s *server) doesPDVExistHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pdv-exists Cerberus DoesPDVExist
	//
	// Checks if PDV data exists in storage.
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

	if err := api.Verify(req); err != nil {
		writeVerifyError(getLogger(r.Context()), w, err)
		return
	}

	exists, err := s.s.DoesPDVExist(r.Context(), req.Address)
	if err != nil {
		writeInternalError(getLogger(r.Context()), w, err.Error())
		return
	}

	writeOK(w, http.StatusOK, api.DoesPDVExistResponse{PDVExists: exists})
}
