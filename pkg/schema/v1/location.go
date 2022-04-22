package schema

import (
	"github.com/Decentr-net/cerberus/pkg/schema/types"
)

// Location is user's geolocation.
type Location struct {
	types.Timestamp

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	RequestedBy *types.Source `json:"requestedBy"`
}

// Type ...
func (Location) Type() types.Type {
	return types.PDVLocationType
}

// Validate ...
func (d Location) Validate() bool {
	if d.Latitude < -90 || d.Latitude > 90 {
		return false
	}

	if d.Longitude < -180 || d.Longitude > 180 {
		return false
	}

	if d.RequestedBy != nil && !d.RequestedBy.Validate() {
		return false
	}

	return d.Timestamp.Validate()
}

// MarshalJSON ...
func (d Location) MarshalJSON() ([]byte, error) {
	return types.MarshalPDVData(d)
}
