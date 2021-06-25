package schema

import "github.com/Decentr-net/cerberus/internal/schema/types"

const (
	maxAdvertiserLength      = 20
	maxAdvertiserNameLength  = 100
	maxAdvertiserValueLength = 2048 // 2Kb
)

// AdvertiserID is id for advertiser..
type AdvertiserID struct {
	Advertiser string `json:"advertiser"`
	Name       string `json:"name"`
	Value      string `json:"value"`
}

// Type ...
func (AdvertiserID) Type() types.Type {
	return types.PDVAdvertiserIDType
}

// Validate ...
func (d AdvertiserID) Validate() bool {
	if d.Advertiser == "" || d.Name == "" || d.Value == "" {
		return false
	}

	if len(d.Advertiser) > maxAdvertiserLength ||
		len(d.Name) > maxAdvertiserNameLength ||
		len(d.Value) > maxAdvertiserValueLength {
		return false
	}

	return true
}

// MarshalJSON ...
func (d AdvertiserID) MarshalJSON() ([]byte, error) {
	return types.MarshalPDVData(d)
}
