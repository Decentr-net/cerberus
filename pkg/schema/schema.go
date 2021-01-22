// Package schema provides schemas and validation functions for it.
package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// PDVVersion represents version.
type PDVVersion string

const (
	// PDVV1 ...
	PDVV1 PDVVersion = "v1"
)

// TypeVersion represents version of type.
type TypeVersion string

const (
	// CookieV1 ...
	CookieV1 TypeVersion = "v1"
)

const (
	// PDVCookieType ...
	PDVCookieType PDVType = "cookie"
	// PDVLoginCookieType ...
	PDVLoginCookieType PDVType = "login_cookie"
)
const (
	// PDVDataSizeLimit is limit to PDVData's size.
	PDVDataSizeLimit = 8 * 1024
)

// PDVType represents data type.
type PDVType string

// nolint: gochecknoglobals
var (
	pdvObjectSchemes = map[PDVVersion]reflect.Type{
		PDVV1: reflect.TypeOf(PDVObjectV1{}),
	}

	pdvDataSchemes = map[PDVType]map[TypeVersion]reflect.Type{
		PDVCookieType: {
			CookieV1: reflect.TypeOf(PDVDataCookieV1{}),
		},
		PDVLoginCookieType: {
			CookieV1: reflect.TypeOf(PDVDataLoginCookieV1{}),
		},
	}
)

// PDV is main data object.
type PDV struct {
	Version PDVVersion `json:"version"`

	PDV []PDVObject `json:"pdv"`
}

// PDVObject is interface for all versions objects.
type PDVObject interface {
	Validate
}

// PDVObjectMetaV1 is PDVObjectV1 meta data.
type PDVObjectMetaV1 struct {
	// Website information
	Host string `json:"domain"`
	Path string `json:"path"`
}

// PDVObjectV1 is PDVObject implementation with v1 version.
type PDVObjectV1 struct {
	PDVObjectMetaV1

	Data []PDVData `json:"data"`
}

// PDVDataMeta contains common information about data.
type PDVDataMeta struct {
	Type    PDVType     `json:"type"`
	Version TypeVersion `json:"version"`
}

// PDVData is interface for all data types.
type PDVData interface {
	Validate

	Type() PDVType
	Version() TypeVersion
}

// PDVDataCookieV1 is PDVData implementation for Cookies(according to https://developer.chrome.com/extensions/cookies) with version v1.
type PDVDataCookieV1 struct {
	Name           string `json:"name"`
	Value          string `json:"value"`
	Domain         string `json:"domain"`
	Path           string `json:"path"`
	SameSite       string `json:"same_site"`
	HostOnly       bool   `json:"host_only"`
	Secure         bool   `json:"secure"`
	ExpirationDate uint64 `json:"expiration_date,omitempty"`
}

// PDVDataLoginCookieV1 is the same as PDVDataCookieV1 but with different type.
type PDVDataLoginCookieV1 PDVDataCookieV1

// UnmarshalJSON ...
func (p *PDV) UnmarshalJSON(b []byte) error {
	var i struct {
		Version PDVVersion `json:"version"`

		PDV []json.RawMessage `json:"pdv"`
	}

	if err := json.Unmarshal(b, &i); err != nil {
		return fmt.Errorf("failed to unmarshal PDV meta: %w", err)
	}

	t, ok := pdvObjectSchemes[i.Version]
	if !ok {
		return errors.New("unknown version of object")
	}

	p.Version = i.Version

	for _, v := range i.PDV {
		o := reflect.New(t).Interface()

		if err := json.Unmarshal(v, o); err != nil {
			return err
		}

		p.PDV = append(p.PDV, o.(PDVObject))
	}

	return nil
}

// UnmarshalJSON ...
func (o *PDVObjectV1) UnmarshalJSON(b []byte) error {
	var i struct {
		PDVObjectMetaV1

		PDVData []json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	*o = PDVObjectV1{
		PDVObjectMetaV1: i.PDVObjectMetaV1,
		Data:            make([]PDVData, len(i.PDVData)),
	}

	for i, v := range i.PDVData {
		if len(v) > PDVDataSizeLimit {
			return errors.New("pdv data is too big")
		}

		var m PDVDataMeta
		if err := json.Unmarshal(v, &m); err != nil {
			return fmt.Errorf("failed to unmarshal PDV data meta: %w", err)
		}

		t, ok := pdvDataSchemes[m.Type][m.Version]
		if !ok {
			return fmt.Errorf("unknown pdv data: %s %s", m.Type, m.Version)
		}

		d := reflect.New(t).Interface().(PDVData) // nolint:errcheck

		if err := json.Unmarshal(v, d); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}

		o.Data[i] = d
	}

	return nil
}

// Version ...
func (PDVDataCookieV1) Version() TypeVersion {
	return CookieV1
}

// Type ...
func (PDVDataCookieV1) Type() PDVType {
	return PDVCookieType
}

// Version ...
func (PDVDataLoginCookieV1) Version() TypeVersion {
	return CookieV1
}

// Type ...
func (PDVDataLoginCookieV1) Type() PDVType {
	return PDVLoginCookieType
}

// MarshalJSON ...
func (d PDVDataCookieV1) MarshalJSON() ([]byte, error) { // nolint:gocritic
	type T PDVDataCookieV1
	v := struct {
		PDVDataMeta
		T
	}{
		PDVDataMeta: PDVDataMeta{
			Version: d.Version(),
			Type:    d.Type(),
		},
		T: T(d),
	}

	return json.Marshal(v)
}

// MarshalJSON ...
func (d PDVDataLoginCookieV1) MarshalJSON() ([]byte, error) { // nolint:gocritic
	type T PDVDataCookieV1
	v := struct {
		PDVDataMeta
		T
	}{
		PDVDataMeta: PDVDataMeta{
			Version: d.Version(),
			Type:    d.Type(),
		},
		T: T(d),
	}

	return json.Marshal(v)
}
