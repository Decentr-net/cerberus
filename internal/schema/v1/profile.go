package schema

import (
	"encoding/json"
	"fmt"
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
	FirstName string       `json:"first_name"`
	LastName  string       `json:"last_name"`
	Bio       string       `json:"bio"`
	Gender    types.Gender `json:"gender"`
	Avatar    string       `json:"avatar"`
	Birthday  time.Time    `json:"birthday"`
}

// Type ...
func (Profile) Type() types.Type {
	return types.PDVProfileType
}

// MarshalJSON ...
func (d Profile) MarshalJSON() ([]byte, error) { // nolint: gocritic
	return types.MarshalPDVData(d)
}

// UnmarshalJSON ...
func (d *Profile) UnmarshalJSON(b []byte) error {
	var p struct {
		FirstName string       `json:"first_name"`
		LastName  string       `json:"last_name"`
		Bio       string       `json:"bio"`
		Gender    types.Gender `json:"gender"`
		Avatar    string       `json:"avatar"`
		Birthday  string       `json:"birthday"`
	}

	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}

	birthday, err := time.Parse(types.DateFormat, p.Birthday)
	if err != nil {
		return fmt.Errorf("failed to parse birthday: %w", err)
	}

	*d = Profile{
		FirstName: p.FirstName,
		LastName:  p.LastName,
		Bio:       p.Bio,
		Gender:    p.Gender,
		Avatar:    p.Avatar,
		Birthday:  birthday,
	}

	return nil
}

// Validate ...
func (d Profile) Validate() bool { // nolint: gocritic
	return types.IsValidGender(d.Gender) &&
		types.IsValidAvatar(d.Avatar) &&
		types.IsValidDate(d.Birthday.Format(types.DateFormat)) &&
		utf8.RuneCountInString(d.FirstName) <= maxFirstNameLength &&
		utf8.RuneCountInString(d.LastName) <= maxLastNameLength
}
