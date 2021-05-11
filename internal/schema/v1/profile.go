package schema

import (
	"time"
	"unicode/utf8"

	"github.com/Decentr-net/cerberus/internal/schema/types"
)

const (
	maxFirstNameLength = 64
	maxLastNameLength  = 64
)

// Profile is PDVData implementation for profile's data.
type Profile struct {
	FirstName string       `json:"firstName"`
	LastName  string       `json:"lastName"`
	Bio       string       `json:"bio"`
	Gender    types.Gender `json:"gender"`
	Avatar    string       `json:"avatar"`
	Birthday  types.Date   `json:"birthday"`
}

// Type ...
func (Profile) Type() types.Type {
	return types.PDVProfileType
}

// MarshalJSON ...
func (d Profile) MarshalJSON() ([]byte, error) { // nolint: gocritic
	return types.MarshalPDVData(d)
}

// Validate ...
func (d Profile) Validate() bool { // nolint: gocritic
	return types.IsValidGender(d.Gender) &&
		types.IsValidAvatar(d.Avatar) &&
		utf8.RuneCountInString(d.FirstName) <= maxFirstNameLength &&
		utf8.RuneCountInString(d.LastName) <= maxLastNameLength &&
		d.Birthday.Year() > 1900 && d.Birthday.Year() < time.Now().Year()
}