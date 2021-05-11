package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfile_Validate(t *testing.T) {
	tt := []struct {
		name  string
		p     Profile
		valid bool
	}{
		{
			name: "valid",
			p: Profile{
				FirstName: "First",
				LastName:  "Last",
				Bio:       "Some BIO",
				Gender:    "male",
				Avatar:    "https://decentr.xyz/avatar.jpeg",
				Birthday:  mustDate("1990-01-01"),
			},
			valid: true,
		},
		{
			name: "long first name",
			p: Profile{
				FirstName: "VeryLongFirstNameVeryLongFirstNameVeryLongFirstNameVeryLongFirstN",
				LastName:  "Last",
				Bio:       "Some BIO",
				Gender:    "male",
				Avatar:    "https://decentr.xyz/avatar.jpeg",
				Birthday:  mustDate("1990-01-01"),
			},
			valid: false,
		},
		{
			name: "long last name",
			p: Profile{
				FirstName: "First",
				LastName:  "VeryLongLastNameVeryLongLastNameVeryLongLastNameVeryLongLastNameV",
				Bio:       "Some BIO",
				Gender:    "male",
				Avatar:    "https://decentr.xyz/avatar.jpeg",
				Birthday:  mustDate("1990-01-01"),
			},
			valid: false,
		},
		{
			name: "invalid avatar",
			p: Profile{
				FirstName: "First",
				LastName:  "Last",
				Bio:       "Some BIO",
				Gender:    "male",
				Avatar:    "ftp://decentr.xyz/avatar.jpeg",
				Birthday:  mustDate("1990-01-01"),
			},
			valid: false,
		},
		{
			name: "invalid gender",
			p: Profile{
				FirstName: "First",
				LastName:  "Last",
				Bio:       "Some BIO",
				Gender:    "coolguy",
				Avatar:    "https://decentr.xyz/avatar.jpeg",
				Birthday:  mustDate("1990-01-01"),
			},
			valid: false,
		},
		{
			name: "invalid birthday",
			p: Profile{
				FirstName: "First",
				LastName:  "Last",
				Bio:       "Some BIO",
				Gender:    "female",
				Avatar:    "https://decentr.xyz/avatar.jpeg",
				Birthday:  mustDate("0010-01-01"),
			},
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.valid, tc.p.Validate())
		})
	}
}
