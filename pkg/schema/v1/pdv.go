// Package schema implements schema for version v2.
package schema

import (
	"encoding/json"
	"reflect"

	"github.com/Decentr-net/cerberus/pkg/schema/types"
)

// Version ...
const Version types.Version = "v1"

var _ types.PDV = PDV{}

var dataSchemes = types.TypeMapper{ // nolint:gochecknoglobals
	types.PDVAdvertiserIDType:  reflect.TypeOf(AdvertiserID{}),
	types.PDVCookieType:        reflect.TypeOf(Cookie{}),
	types.PDVLocationType:      reflect.TypeOf(Location{}),
	types.PDVSearchHistoryType: reflect.TypeOf(SearchHistory{}),
	types.PDVProfileType:       reflect.TypeOf(Profile{}),
}

// PDV is PDVObject implementation with v2 version.
type PDV []types.Data

// Version returns version of PDV.
func (PDV) Version() types.Version {
	return Version
}

// Data returns slice of data.
func (o PDV) Data() []types.Data {
	return o
}

// UnmarshalJSON ...
func (o *PDV) UnmarshalJSON(b []byte) error {
	var data []json.RawMessage

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	out := make([]types.Data, len(data))

	for i, v := range data {
		d, err := dataSchemes.UnmarshalPDVData(v)
		if err != nil {
			return err
		}

		out[i] = d
	}

	*o = out

	return nil
}

// MarshalJSON ...
func (o PDV) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Version types.Version `json:"version"`
		PDV     interface{}   `json:"pdv"`
	}{
		Version: o.Version(),
		PDV:     o.Data(),
	})
}

// Validate ...
func (o PDV) Validate() bool {
	for _, v := range o {
		if !v.Validate() {
			return false
		}
	}
	return len(o) > 0
}

// GetInvalidPDV returns indicies of invalid PDV.
func GetInvalidPDV(b []byte) ([]int, error) {
	var data []json.RawMessage

	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	out := make([]int, 0, len(data))

	for i, v := range data {
		pdv, err := dataSchemes.UnmarshalPDVData(v)
		if err != nil || !pdv.Validate() {
			out = append(out, i)
		}
	}

	return out, nil
}
