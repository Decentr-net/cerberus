// Package schema provides schemas and validation functions for it.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/Decentr-net/cerberus/pkg/schema/types"
	v1 "github.com/Decentr-net/cerberus/pkg/schema/v1"
)

// nolint
type (
	PDV     = types.PDV
	Data    = types.Data
	Type    = types.Type
	Source  = types.Source
	Version = types.Version
)

// nolint
const (
	V1 = v1.Version
)

// nolint
const (
	PDVAdvertiserIDType  = types.PDVAdvertiserIDType
	PDVCookieType        = types.PDVCookieType
	PDVLocationType      = types.PDVLocationType
	PDVProfileType       = types.PDVProfileType
	PDVSearchHistoryType = types.PDVSearchHistoryType
)

// nolint
type (
	V1AdvertiserID  = v1.AdvertiserID
	V1Cookie        = v1.Cookie
	V1Location      = v1.Location
	V1Profile       = v1.Profile
	V1SearchHistory = v1.SearchHistory
)

// nolint: gochecknoglobals
var (
	pdvObjectSchemes = map[Version]PDV{
		V1: v1.PDV{},
	}
)

var _ types.PDV = PDVWrapper{}

// PDVWrapper is wrapper for PDV object.
//
// It's very usable for composing into request or response.
type PDVWrapper struct {
	Device string
	pdv    types.PDV
}

// NewPDVWrapper create a new PDV wrapper.
func NewPDVWrapper(device string, pdv types.PDV) PDVWrapper {
	return PDVWrapper{
		Device: device,
		pdv:    pdv,
	}
}

// MarshalJSON ...
func (p PDVWrapper) MarshalJSON() ([]byte, error) {
	if p.pdv == nil {
		return nil, errors.New("pdv is not specified")
	}

	var data []Data
	for _, d := range p.Data() {
		if d.Type() != PDVCookieType {
			data = append(data, d)
		}
	}

	return json.Marshal(struct {
		Device  string  `json:"device"`
		Version Version `json:"version"`
		PDV     []Data  `json:"pdv"`
	}{
		PDV:     data,
		Device:  p.Device,
		Version: p.Version(),
	})
}

// UnmarshalJSON ...
func (p *PDVWrapper) UnmarshalJSON(b []byte) error {
	var i struct {
		Version Version         `json:"version"`
		Device  string          `json:"device"`
		PDV     json.RawMessage `json:"pdv"`
	}

	if err := json.Unmarshal(b, &i); err != nil {
		return fmt.Errorf("failed to unmarshal PDV meta: %w", err)
	}

	t, ok := pdvObjectSchemes[i.Version]
	if !ok {
		return errors.New("unknown version of object")
	}

	p.pdv = reflect.New(reflect.TypeOf(t)).Interface().(PDV) // nolint: errcheck
	p.Device = i.Device

	if i.PDV == nil {
		return nil
	}
	if err := json.Unmarshal(i.PDV, p.pdv); err != nil {
		return err
	}

	return nil
}

// Validate returns true if pdv is valid.
func (p PDVWrapper) Validate() bool {
	switch p.Device {
	case "", "ios", "android", "desktop":
		return p.pdv.Validate()
	}
	return false
}

// Version ...
func (p PDVWrapper) Version() Version {
	return p.pdv.Version()
}

// Data ...
func (p PDVWrapper) Data() []Data {
	return p.pdv.Data()
}

// GetInvalidPDV return indices  of invalid pdv.
func GetInvalidPDV(b []byte) ([]int, error) {
	var i struct {
		Version Version `json:"version"`

		PDV json.RawMessage `json:"pdv"`
	}

	if err := json.Unmarshal(b, &i); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PDV meta: %w", err)
	}

	switch i.Version {
	case V1:
		return v1.GetInvalidPDV(i.PDV)
	default:
		return nil, fmt.Errorf("invalid version")
	}
}
