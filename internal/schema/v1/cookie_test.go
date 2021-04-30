package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Decentr-net/cerberus/internal/schema/types"
)

func TestPDVDataCookie_Validate(t *testing.T) {
	tt := []struct {
		name  string
		c     Cookie
		valid bool
	}{
		{
			name: "valid",
			c: Cookie{
				Source: types.Source{
					Host: "https://decentr.xyz",
					Path: "/?something#",
				},
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
			c: Cookie{
				Source: types.Source{Host: "https://decentr.xyz"},
				Name:   "name",
				Value:  "value",
			},
			valid: true,
		},
		{
			name: "without name",
			c: Cookie{
				Source: types.Source{Host: "https://decentr.xyz"},
				Value:  "value",
			},
			valid: false,
		},
		{
			name: "without value",
			c: Cookie{
				Source: types.Source{Host: "https://decentr.xyz"},
				Name:   "name",
			},
			valid: false,
		},
		{
			name: "without host",
			c: Cookie{
				Name:  "name",
				Value: "value",
			},
			valid: false,
		},
		{
			name: "invalid host",
			c: Cookie{
				Source: types.Source{Host: "abc"},
				Name:   "name",
				Value:  "value",
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
