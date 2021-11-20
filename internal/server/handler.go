package server

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/go-chi/chi"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/Decentr-net/go-api"
	logging "github.com/Decentr-net/logrus/context"

	"github.com/Decentr-net/cerberus/internal/entities"
	"github.com/Decentr-net/cerberus/internal/schema"
	"github.com/Decentr-net/cerberus/internal/service"
)

// SavePDVResponse ...
// swagger:model SavePDVResponse
type SavePDVResponse struct {
	ID uint64 `json:"id"`
}

// SaveImageResponse ...
// swagger:model SaveImageResponse
type SaveImageResponse struct {
	HD    string `json:"hd"`
	Thumb string `json:"thumb"`
}

// saveImageHandler resizes and saves the given message into storage.
func (s *server) saveImageHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /images Image Save
	//
	// Resizes (1920x1080, 480x270) and saves images
	// ---
	// security:
	// - public_key: []
	//   signature: []
	// produces:
	// - application/json
	// consumes:
	// - multipart/form-data
	// responses:
	//   '200':
	//     description: image successfully resized and saved as HD (1920x1080) and thumbnail (480x270)
	//     schema:
	//       "$ref": "#/definitions/SaveImageResponse"
	//   '500':
	//      description: internal server error
	//      schema:
	//        "$ref": "#/definitions/Error"

	if err := api.Verify(r); err != nil {
		api.WriteVerifyError(r.Context(), w, err)
		return
	}

	owner, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to decode owner address: %s", err.Error()))
		return
	}

	hdPath, thumbPath, err := s.s.SaveImage(r.Context(), r.Body, owner.String())
	if err != nil {
		if errors.Is(err, service.ErrImageInvalidFormat) {
			api.WriteError(w, http.StatusBadRequest, "request body is not an image")
			return
		}

		api.WriteInternalErrorf(r.Context(), w, "failed to save image: %s", err.Error())
		return
	}

	api.WriteOK(w, http.StatusOK, SaveImageResponse{
		HD:    hdPath,
		Thumb: thumbPath,
	})
}

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
		api.WriteVerifyError(r.Context(), w, err)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to read body: %s", err.Error()))
		return
	}
	r.Body.Close() // nolint:errcheck,gosec

	var p schema.PDVWrapper
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

	if l := uint16(len(p.Data())); l < s.minPDVCount || l > s.maxPDVCount {
		if l != 1 || p.Data()[0].Type() != schema.PDVProfileType {
			api.WriteError(w, http.StatusBadRequest, "forbidden pdv count")
			return
		}
	}

	owner, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to decode owner address: %s", err.Error()))
		return
	}

	if s.savePDVThrottler.Throttle(owner.String()) {
		api.WriteError(w, http.StatusTooManyRequests,
			fmt.Sprintf("too many requests for %s", owner.String()))
		return
	}

	id, meta, err := s.s.SavePDV(r.Context(), p, owner)
	if err != nil {
		api.WriteInternalErrorf(r.Context(), w, "failed to save pdv: %s", err.Error())
		return
	}

	s.pdvMetaCache.Add(getCacheKey(owner.String(), id), meta)
	s.savePDVThrottler.Reset(owner.String())

	api.WriteOK(w, http.StatusCreated, SavePDVResponse{ID: id})
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

	if err := api.Verify(r); err != nil {
		api.WriteVerifyError(r.Context(), w, err)
		return
	}

	owner, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
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

	var m *entities.PDVMeta
	if v, ok := s.pdvMetaCache.Get(getCacheKey(owner, id)); ok {
		logging.GetLogger(r.Context()).WithField("key", getCacheKey(owner, id)).Debug("meta found in cache")
		m = v.(*entities.PDVMeta) // nolint
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

// getProfilesHandler returns profiles.
func (s *server) getProfilesHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /profiles Profile GetProfiles
	//
	// Get profiles
	//
	// Returns profiles by addresses
	//
	// ---
	// parameters:
	// - name: address
	//   description: profile address
	//   in: query
	//   required: true
	//   example: decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz,decentr1u9slwz3sje8j94ccpwlslflg0506yc8y2ylmtz
	//   schema:
	//      type: array
	//      items:
	//          type: string
	// responses:
	//   '200':
	//     description: metadata of pdv
	//     schema:
	//      type: array
	//      items:
	//          "$ref": "#/definitions/APIProfile"
	//   '500':
	//     description: internal server error
	//     schema:
	//       "$ref": "#/definitions/Error"

	address := strings.Split(r.URL.Query().Get("address"), ",")

	var requestedBy string
	if r.Header.Get(api.PublicKeyHeader) != "" {
		if err := api.Verify(r); err != nil {
			api.WriteVerifyError(r.Context(), w, err)
			return
		}

		owner, err := getAddressFromPubKey(r.Header.Get(api.PublicKeyHeader))
		if err != nil {
			api.WriteError(w, http.StatusBadRequest, fmt.Sprintf("failed to decode owner address: %s", err.Error()))
			return
		}

		requestedBy = owner.String()
	}

	for _, v := range address {
		if !isOwnerValid(v) {
			api.WriteError(w, http.StatusBadRequest, "invalid address")
			return
		}
	}

	pp, err := s.s.GetProfiles(r.Context(), address)
	if err != nil {
		api.WriteInternalError(r.Context(), w, fmt.Sprintf("failed to get profiles: %s", err.Error()))
		return
	}

	out := make([]Profile, len(pp))
	for i, v := range pp {
		p := Profile{
			Address:   v.Address,
			FirstName: v.FirstName,
			LastName:  v.LastName,
			Bio:       v.Bio,
			Gender:    v.Gender,
			Avatar:    v.Avatar,
			Birthday:  v.Birthday.Format(dateFormat),
			CreatedAt: v.CreatedAt.Unix(),
		}

		if requestedBy == p.Address {
			p.Emails = v.Emails
		}

		out[i] = p
	}

	api.WriteOK(w, http.StatusOK, out)
}

// getRewardsConfigHandler returns rewards config.
func (s *server) getRewardsConfigHandler(w http.ResponseWriter, _ *http.Request) {
	// swagger:operation GET /configs/rewards Configs GetRewardsConfig
	//
	// Get rewards config
	//
	// Returns rewards config.
	//
	// ---
	// responses:
	//   '200':
	//     description: rewards config
	//     schema:
	//       "$ref": "#/definitions/ObjectTypes"
	//   '500':
	//     description: internal server error
	//     schema:
	//       "$ref": "#/definitions/Error"

	api.WriteOK(w, http.StatusOK, s.s.GetRewardsMap())
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
