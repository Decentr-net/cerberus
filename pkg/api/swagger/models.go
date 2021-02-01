package swagger

import "github.com/Decentr-net/cerberus/pkg/schema"

// swagger:model PDV
type PDVInterface interface {
	// discriminator: true
	// swagger:name version
	Version() schema.PDVVersion
}

// PDV is main data object.
// swagger:model v1
type PDV struct {
	// swagger:allOf v1
	PDVInterface

	PDV []PDVObjectV1 `json:"pdv"`
}

func (PDV) Version() schema.PDVVersion {
	return schema.PDVV1
}

// PDVObjectV1 is PDVObject implementation with v1 version.
// swagger:model PDVObjectV1
type PDVObjectV1 struct {
	Data []PDVData `json:"data"`

	schema.PDVObjectMetaV1
}

// PDVData is interface for all data types.
// swagger:model PDVData
type PDVData interface {
	// discriminator: true
	// swagger:name type
	Type() schema.PDVType
}

// PDVDataCookie is PDVData implementation for Cookies(according to https://developer.chrome.com/extensions/cookies).
// swagger:model cookie
type PDVDataCookie struct {
	// swagger:allOf cookie
	PDVData

	schema.PDVDataCookie
}

// PDVDataLoginCookie is the same as PDVDataCookie but with different type.
// swagger:model login_cookie
type PDVDataLoginCookie struct {
	// swagger:allOf login_cookie
	PDVData

	schema.PDVDataLoginCookie
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
