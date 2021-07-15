package swagger

import (
	"github.com/Decentr-net/cerberus/internal/schema"
	"github.com/Decentr-net/cerberus/internal/schema/types"
	v1 "github.com/Decentr-net/cerberus/internal/schema/v1"
)

// swagger:model PDV
type PDVInterface interface {
	// discriminator: true
	// swagger:name version
	Version() schema.Version
}

// PDVV1 is main data object.
// swagger:model v1
type PDVV1 struct {
	// swagger:allOf v1
	PDVInterface

	PDV []DataV1 `json:"pdv"`
}

// DataV1 is interface for all data types.
// swagger:model DataV1
type DataV1 interface {
	// discriminator: true
	// swagger:name type
	TypeV1() types.Type
}

// AdvertiserIDV1 contains id for an advertiser (e.g google, facebook).
// swagger:model advertiserId
type AdvertiserIDV1 struct {
	// swagger:allOf advertiserId
	DataV1

	v1.AdvertiserID
}

// CookieV1 is PDVData implementation for Cookies(according to https://developer.chrome.com/extensions/cookies).
// swagger:model cookie
type CookieV1 struct {
	// swagger:allOf cookie
	DataV1

	v1.Cookie
}

// LocationV1 contains user's geolocation at a time.
// swagger:model location
type LocationV1 struct {
	// swagger:allOf location
	DataV1

	v1.Location
}

// ProfileV1 is profile data.
// swagger:model profile
type ProfileV1 struct {
	// swagger:allOf profile
	DataV1

	v1.Profile
}

// SearchHistoryV1 contains user's search request.
// swagger:model searchHistory
type SearchHistoryV1 struct {
	// swagger:allOf searchHistory
	DataV1

	v1.SearchHistory
}

// PDVMeta contains info about PDV.
// swagger:model PDVMeta
type PDVMeta struct {
	// ObjectTypes represents how much certain pdv data pdv contains.
	ObjectTypes ObjectTypes `json:"object_types"`
	Reward      uint64      `json:"reward"`
}

// ObjectTypes contains count of each pdv type in batch.
type ObjectTypes struct {
	AdvertiserID    uint16 `json:"advertiserId"`
	Cookie          uint16 `json:"cookie"`
	Location        uint16 `json:"location"`
	Profile         uint16 `json:"profile"`
	SearchHistoryV1 uint16 `json:"searchHistory"`
}
