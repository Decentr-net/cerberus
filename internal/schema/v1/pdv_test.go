package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Decentr-net/cerberus/internal/schema/types"
)

func TestPDV_Validate(t *testing.T) {
	require.True(t, PDV{
		Cookie{
			Timestamp: types.Timestamp{Time: time.Now()},
			Source:    types.Source{Host: "https://decentr.xyz"},
			Name:      "cookie",
			Value:     "value",
		},
		Profile{
			FirstName: "First",
			LastName:  "Last",
			Emails:    []string{"test@decentr.xyz"},
			Gender:    types.GenderMale,
			Avatar:    "https://decentr.xyz/avatar.jpeg",
			Birthday:  mustDate("1990-01-01"),
		},
	}.Validate())
}

func TestPDV_Validate_invalid(t *testing.T) {
	require.False(t, PDV{}.Validate())

	require.False(t, PDV{
		Cookie{
			Source: types.Source{Host: "https://decentr.xyz"},
			Name:   "cookie",
		},
	}.Validate())
}

func mustDate(s string) *types.Date {
	var d types.Date

	if err := d.UnmarshalJSON([]byte(s)); err != nil {
		panic(err)
	}

	return &d
}
