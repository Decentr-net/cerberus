package schema

import "github.com/Decentr-net/cerberus/internal/schema/types"

const (
	maxAdvertiserLength   = 20
	maxAdvertiserIDLength = 100
)

// AdvertiserID is id for advertiser..
type AdvertiserID struct {
	Advertiser string `json:"advertiser"`
	ID         string `json:"id"`
}

// Type ...
func (AdvertiserID) Type() types.Type {
	return types.PDVAdvertiserIDType
}

// Validate ...
func (d AdvertiserID) Validate() bool {
	if d.Advertiser == "" || d.ID == "" {
		return false
	}

	if len(d.Advertiser) > maxAdvertiserLength || len(d.ID) > maxAdvertiserIDLength {
		return false
	}

	return true
}

// MarshalJSON ...
func (d AdvertiserID) MarshalJSON() ([]byte, error) {
	return types.MarshalPDVData(d)
}
