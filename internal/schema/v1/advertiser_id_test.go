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
				ID:         "12345",
			},
			valid: true,
		},
		{
			name: "empty_advertiser",
			d: AdvertiserID{
				Advertiser: "",
				ID:         "12345",
			},
			valid: false,
		},
		{
			name: "empty_id",
			d: AdvertiserID{
				Advertiser: "advertiser",
				ID:         "",
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
