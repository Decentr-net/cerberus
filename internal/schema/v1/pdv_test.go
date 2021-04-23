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
			Source: types.Source{Host: "https://decentr.xyz"},
			Name:   "cookie",
			Value:  "value",
		},
		Profile{
			FirstName: "First",
			LastName:  "Last",
			Gender:    types.GenderMale,
			Avatar:    "https://decentr.xyz/avatar.jpeg",
			Birthday:  time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
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
