package schema

import (
	"github.com/Decentr-net/cerberus/internal/schema/types"
)

// Cookie is PDVData implementation for Cookies(according to https://developer.chrome.com/extensions/cookies).
type Cookie struct {
	types.Timestamp

	Source types.Source `json:"source"`

	Name           string `json:"name"`
	Value          string `json:"value"`
	Domain         string `json:"domain"`
	Path           string `json:"path"`
	SameSite       string `json:"sameSite"`
	HostOnly       bool   `json:"hostOnly"`
	Secure         bool   `json:"secure"`
	ExpirationDate uint64 `json:"expirationDate,omitempty"`
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

	return d.Source.Validate() && d.Timestamp.Validate()
}

// MarshalJSON ...
func (d Cookie) MarshalJSON() ([]byte, error) { // nolint:gocritic
	return types.MarshalPDVData(d)
}
