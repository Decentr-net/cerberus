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
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tomasen/realip"

	"github.com/Decentr-net/cerberus/internal/service"
	"github.com/Decentr-net/cerberus/pkg/api"
	"github.com/Decentr-net/cerberus/pkg/schema"
)

type metaPDVData struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

type serverPDV struct {
	UserData schema.PDV  `json:"user_data"`
	MetaData metaPDVData `json:"calculated_data"`
}

// savePDVHandler encrypts and puts PDV data into storage.
func (s *server) savePDVHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /pdv Cerberus SavePDV
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
	//       "$ref": "#/definitions/SavePDVResponse"
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

	address, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate address")
	}

	filepath := fmt.Sprintf("%s-%s", address, hex.EncodeToString(digest))

	spdv, err := json.Marshal(serverPDV{
		UserData: p,
		MetaData: metaPDVData{
			IP:        realip.FromRequest(r),
			UserAgent: r.UserAgent(),
		},
	})
	if err != nil {
		writeInternalError(getLogger(r.Context()), w, fmt.Sprintf("failed to marshal modified PDV: %s", err.Error()))
		return
	}

	if err := s.s.SavePDV(r.Context(), spdv, filepath); err != nil {
		writeInternalError(getLogger(r.Context()), w, err.Error())
		return
	}

	s.pdvExistenceCache.Add(filepath, true)

	writeOK(w, http.StatusCreated, api.SavePDVResponse{Address: filepath})
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

func getAddressFromPubKey(k string) (string, error) {
	var pk secp256k1.PubKeySecp256k1
	b, _ := hex.DecodeString(k)
	if err := cdc.UnmarshalBinaryBare(b, &pk); err != nil {
		return "", err
	}
	return hex.EncodeToString(pk.Address()), nil
}
