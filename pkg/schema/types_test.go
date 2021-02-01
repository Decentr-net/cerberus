package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDVDataCookieV1_Validate(t *testing.T) {
	tt := []struct {
		name  string
		c     PDVDataCookie
		valid bool
	}{
		{
			name: "valid",
			c: PDVDataCookie{
				Name:           "name",
				Value:          "value",
				Domain:         "decentr.net",
				HostOnly:       true,
				Path:           "*",
				Secure:         true,
				SameSite:       "*",
				ExpirationDate: 123413412,
			},
			valid: true,
		},
		{
			name: "valid minimal",
			c: PDVDataCookie{
				Name:  "name",
				Value: "value",
			},
			valid: true,
		},
		{
			name: "without name",
			c: PDVDataCookie{
				Value: "value",
			},
			valid: false,
		},
		{
			name: "without value",
			c: PDVDataCookie{
				Name: "name",
			},
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, tc.c.Validate())
		})
	}
}

func TestPDVType_UnmarshalText(t *testing.T) {
	var p PDVType
	require.NoError(t, p.UnmarshalText([]byte(PDVCookieType)))
	require.Error(t, p.UnmarshalText([]byte("wrong")))
}
