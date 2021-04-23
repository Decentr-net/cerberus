package schema

import (
	"github.com/Decentr-net/cerberus/internal/schema/types"
)

// Cookie is PDVData implementation for Cookies(according to https://developer.chrome.com/extensions/cookies).
type Cookie struct {
	Source types.Source `json:"source"`

	Name           string `json:"name"`
	Value          string `json:"value"`
	Domain         string `json:"domain"`
	Path           string `json:"path"`
	SameSite       string `json:"same_site"`
	HostOnly       bool   `json:"host_only"`
	Secure         bool   `json:"secure"`
	ExpirationDate uint64 `json:"expiration_date,omitempty"`
}

// Type ...
func (Cookie) Type() types.Type {
	return types.PDVCookieType
}

// Validate ...
func (d Cookie) Validate() bool { // nolint: gocritic
	if d.Name == "" || d.Value == "" {
		return false
	}

	return d.Source.Validate()
}

// MarshalJSON ...
func (d Cookie) MarshalJSON() ([]byte, error) { // nolint:gocritic
	return types.MarshalPDVData(d)
}

// LoginCookie is the same as PDVDataCookie but with different type.
type LoginCookie Cookie

// Type ...
func (LoginCookie) Type() types.Type {
	return types.PDVLoginCookieType
}

// MarshalJSON ...
func (d LoginCookie) MarshalJSON() ([]byte, error) { // nolint:gocritic
	return types.MarshalPDVData(d)
}

// Validate ...
func (d LoginCookie) Validate() bool { // nolint: gocritic
	return Cookie(d).Validate()
}
