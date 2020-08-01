// Package schema provides schemas and validation functions for it.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

const (
	// PDVv1 ...
	PDVv1 PDVVersion = "v1"
)
const (
	// PDVCookieType ...
	PDVCookieType PDVType = "cookie"
)

// PDVVersion represents version.
type PDVVersion string

// PDVType represents data type.
type PDVType string

// nolint: gochecknoglobals
var (
	pdvObjectSchemes = map[PDVVersion]reflect.Type{
		PDVv1: reflect.TypeOf(PDVObjectV1{}),
	}

	pdvDataSchemes = map[PDVType]map[PDVVersion]reflect.Type{
		PDVCookieType: {
			PDVv1: reflect.TypeOf(PDVDataCookieV1{}),
		},
	}
)

// PDV is main data object.
type PDV struct {
	Version PDVVersion `json:"version"`

	PDV PDVObject `json:"pdv"`
}

// PDVObject is interface for all versions objects.
type PDVObject interface {
	Validate

	Version() PDVVersion
}

// PDVObjectV1 is PDVObject implementation with v1 version.
type PDVObjectV1 struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`

	Data []PDVData `json:"data"`
}

// Version ...
func (o *PDVObjectV1) Version() PDVVersion {
	return PDVv1
}

// PDVDataMeta contains common information about data.
type PDVDataMeta struct {
	PDVVersion PDVVersion `json:"version"`
	PDVType    PDVType    `json:"type"`
}

// PDVData is interface for all data types.
type PDVData interface {
	Validate

	Version() PDVVersion
	Type() PDVType
}

// PDVDataCookieV1 is PDVData implementation for Cookies with version v1.
type PDVDataCookieV1 struct {
	PDVDataMeta

	Name    string `json:"name"`
	Value   string `json:"value"`
	Expires string `json:"expires,omitempty"`
	MaxAge  uint32 `json:"max_age,omitempty"`
	Path    string `json:"path,omitempty"`
	Domain  string `json:"domain,omitempty"`
}

// Version ...
func (d PDVDataMeta) Version() PDVVersion {
	return d.PDVVersion
}

// Type ...
func (d PDVDataMeta) Type() PDVType {
	return d.PDVType
}

// UnmarshalJSON ...
func (p *PDV) UnmarshalJSON(b []byte) error {
	var i struct {
		Version PDVVersion      `json:"version"`
		PDV     json.RawMessage `json:"PDV"`
	}

	if err := json.Unmarshal(b, &i); err != nil {
		return fmt.Errorf("failed to unmarshal PDV meta: %w", err)
	}

	t, ok := pdvObjectSchemes[i.Version]
	if !ok {
		return errors.New("unknown version of object")
	}

	v := reflect.New(t).Interface()
	if err := json.Unmarshal(i.PDV, v); err != nil {
		return err
	}

	p.Version = i.Version
	p.PDV = v.(PDVObject) // nolint

	return nil
}

// UnmarshalJSON ...
func (o *PDVObjectV1) UnmarshalJSON(b []byte) error {
	var i struct {
		IP        string `json:"ip"`
		UserAgent string `json:"user_agent"`

		PDVData []json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	*o = PDVObjectV1{
		IP:        i.IP,
		UserAgent: i.UserAgent,
		Data:      make([]PDVData, len(i.PDVData)),
	}

	for i, v := range i.PDVData {
		var m PDVDataMeta
		if err := json.Unmarshal(v, &m); err != nil {
			return fmt.Errorf("failed to unmarshal PDV data meta: %w", err)
		}

		t, ok := pdvDataSchemes[m.PDVType][m.PDVVersion]
		if !ok {
			return fmt.Errorf("unknown pdv data: %s %s", m.PDVType, m.PDVVersion)
		}

		d := reflect.New(t).Interface().(PDVData) // nolint:errcheck

		if err := json.Unmarshal(v, d); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}

		o.Data[i] = d
	}

	return nil
}
