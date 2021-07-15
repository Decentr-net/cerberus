// Package types contains shared types and constants for schema package.
package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	valid "github.com/asaskevich/govalidator"
)

// Version ...
// swagger:enum Version
type Version string

// Type ...
// swagger:enum Type
type Type string

// nolint
const (
	PDVAdvertiserIDType  Type = "advertiserId"
	PDVCookieType        Type = "cookie"
	PDVProfileType       Type = "profile"
	PDVSearchHistoryType Type = "searchHistory"
	PDVLocationType      Type = "location"
)

const (
	// DataSizeLimit is limit to PDVData's size.
	DataSizeLimit = 8 * 1024
)

// Date in ISO-8601 format (yyyy-mm-dd).
type Date struct {
	time.Time
}

// Timestamp in RFC-3339 format.
type Timestamp struct {
	Time time.Time `json:"timestamp"`
}

// Gender can be male or female.
type Gender string

// nolint
const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// nolint
const DateFormat = "2006-01-02"

// TypeMapper contains rules to decode PDVData.
type TypeMapper map[Type]reflect.Type

// Source contains information about source of pdv.
type Source struct {
	// Domain of website where object was taken
	Host string `json:"host"`
	// Path of website's url where object was taken
	Path string `json:"path"`
}

// Validate ...
type Validate interface {
	Validate() bool
}

// PDV is interface for all versions objects.
type PDV interface {
	Validate

	Version() Version
	Data() []Data
}

// Data is interface for all PDV data types.
type Data interface {
	Validate

	Type() Type
}

// UnmarshalText ...
func (t *Type) UnmarshalText(b []byte) error {
	s := Type(b)
	switch s {
	case PDVAdvertiserIDType, PDVCookieType, PDVLocationType, PDVSearchHistoryType, PDVProfileType:
	default:
		return errors.New("unknown PDVType")
	}
	*t = s
	return nil
}

// MarshalJSON ...
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.Format(DateFormat))), nil
}

// UnmarshalJSON ...
func (d *Date) UnmarshalJSON(b []byte) error {
	t, err := time.Parse(DateFormat, strings.Trim(string(b), `"`))
	if err != nil {
		return err
	}

	*d = Date{t}

	return nil
}

// MarshalPDVData encodes PDVData (with its type).
func MarshalPDVData(data Data) ([]byte, error) {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)

	ff := make([]reflect.StructField, 0, t.NumField()+1)

	for i := 0; i < t.NumField(); i++ {
		ff = append(ff, t.Field(i))
	}

	ff = append(ff, reflect.TypeOf(struct {
		Type Type `json:"type"`
	}{}).Field(0))

	val := reflect.New(reflect.StructOf(ff)).Elem()

	val.FieldByName("Type").SetString(string(data.Type()))
	for i := 0; i < t.NumField(); i++ {
		val.FieldByName(t.Field(i).Name).Set(v.Field(i))
	}

	return json.Marshal(val.Interface())
}

// UnmarshalPDVData decodes b into PDVData object.
func (m TypeMapper) UnmarshalPDVData(b []byte) (Data, error) {
	if len(b) > DataSizeLimit {
		return nil, errors.New("data is too big")
	}

	type T struct {
		Type Type `json:"type"`
	}

	var d T
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PDV Data meta: %w", err)
	}

	t, ok := m[d.Type]
	if !ok {
		return nil, fmt.Errorf("unknown pdv Data: %s", d.Type)
	}

	val := reflect.New(t).Interface().(Data) // nolint:errcheck

	if err := json.Unmarshal(b, val); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Data: %w", err)
	}

	return val, nil
}

// Validate ...
func (s Source) Validate() bool {
	return valid.IsURL(fmt.Sprintf("%s/%s", s.Host, s.Path))
}

// IsValidGender checks if s is a valid gender.
func IsValidGender(s Gender) bool {
	return s == GenderMale || s == GenderFemale
}

// IsValidAvatar checks if avatar url is valid.
func IsValidAvatar(str string) bool {
	if len(str) > 4*1024 {
		return false
	}

	url, err := url.Parse(str)
	if err != nil {
		return false
	}
	return url.Scheme == "http" || url.Scheme == "https"
}

// Validate ...
func (t Timestamp) Validate() bool {
	return !t.Time.IsZero()
}
