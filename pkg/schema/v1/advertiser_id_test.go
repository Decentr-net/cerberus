package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdvertiserID_Validate(t *testing.T) {
	tt := []struct {
		name  string
		d     AdvertiserID
		valid bool
	}{
		{
			name: "valid",
			d: AdvertiserID{
				Advertiser: "advertiser",
				Name:       "Name1",
				Value:      "Value1",
			},
			valid: true,
		},
		{
			name: "empty_advertiser",
			d: AdvertiserID{
				Advertiser: "",
				Name:       "Name1",
				Value:      "Value1",
			},
			valid: false,
		},
		{
			name: "empty_name",
			d: AdvertiserID{
				Advertiser: "advertiser",
				Name:       "",
				Value:      "Value1",
			},
			valid: false,
		},
		{
			name: "empty_value",
			d: AdvertiserID{
				Advertiser: "advertiser",
				Name:       "Name1",
				Value:      "",
			},
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, tc.d.Validate())
		})
	}
}
