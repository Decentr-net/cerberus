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

// Cookie is PDVData implementation for Cookies(according to https://developer.chrome.com/extensions/cookies).
// swagger:model cookie
type Cookie struct {
	// swagger:allOf cookie
	DataV1

	v1.Cookie
}

// LoginCookieV1 is the same as PDVDataCookie but with different type.
// swagger:model login_cookie
type LoginCookieV1 struct {
	// swagger:allOf login_cookie
	DataV1

	v1.LoginCookie
}

// ProfileV1 is profile data.
// swagger:model profile
type ProfileV1 struct {
	// swagger:allOf profile
	DataV1

	v1.Profile
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
	Cookie      uint16 `json:"cookie"`
	LoginCookie uint16 `json:"login_cookie"`
}
