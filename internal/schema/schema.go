// Package schema provides schemas and validation functions for it.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/Decentr-net/cerberus/internal/schema/types"
	v1 "github.com/Decentr-net/cerberus/internal/schema/v1"
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
	PDVCookieType      = types.PDVCookieType
	PDVLoginCookieType = types.PDVLoginCookieType
	PDVProfileType     = types.PDVProfileType
)

// nolint
type (
	V1Profile     = v1.Profile
	V1Cookie      = v1.Cookie
	V1LoginCookie = v1.LoginCookie
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
	pdv types.PDV
}

// UnmarshalJSON ...
func (p *PDVWrapper) UnmarshalJSON(b []byte) error {
	var i struct {
		Version Version `json:"version"`

		PDV json.RawMessage `json:"pdv"`
	}

	if err := json.Unmarshal(b, &i); err != nil {
		return fmt.Errorf("failed to unmarshal PDV meta: %w", err)
	}

	t, ok := pdvObjectSchemes[i.Version]
	if !ok {
		return errors.New("unknown version of object")
	}

	p.pdv = reflect.New(reflect.TypeOf(t)).Interface().(PDV) // nolint: errcheck

	if i.PDV == nil {
		return nil
	}
	if err := json.Unmarshal(i.PDV, p.pdv); err != nil {
		return err
	}

	return nil
}

// Validate ...
func (p PDVWrapper) Validate() bool {
	return p.pdv.Validate()
}

// Version ...
func (p PDVWrapper) Version() Version {
	return p.pdv.Version()
}

// Data ...
func (p PDVWrapper) Data() []Data {
	return p.pdv.Data()
}
