package swagger

import "github.com/Decentr-net/cerberus/pkg/schema"

func (PDVDataCookie) Type() schema.PDVType {
	return schema.PDVCookieType
}

func (PDVDataLoginCookie) Type() schema.PDVType {
	return schema.PDVLoginCookieType
}
