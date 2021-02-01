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
