package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Decentr-net/cerberus/internal/schema/types"
)

func TestLocation_Validate(t *testing.T) {
	tt := []struct {
		name  string
		l     Location
		valid bool
	}{
		{
			name: "valid",
			l: Location{
				Timestamp:   Timestamp{Time: time.Now()},
				Latitude:    10,
				Longitude:   -10,
				RequestedBy: &types.Source{Host: "decentr.xyz"},
			},
			valid: true,
		},
		{
			name: "valid_zero",
			l: Location{
				Timestamp: Timestamp{Time: time.Now()},
				Latitude:  0,
				Longitude: 0,
			},
			valid: true,
		},
		{
			name: "valid_limit",
			l: Location{
				Timestamp: Timestamp{Time: time.Now()},
				Latitude:  -90,
				Longitude: -180,
			},
			valid: true,
		},
		{
			name: "invalid",
			l: Location{
				Timestamp: Timestamp{Time: time.Now()},
				Latitude:  -91,
				Longitude: -180,
			},
			valid: false,
		},
		{
			name: "invalid time",
			l: Location{
				Latitude:  1,
				Longitude: 18,
			},
			valid: false,
		},
		{
			name: "invalid source",
			l: Location{
				Timestamp:   Timestamp{Time: time.Now()},
				Latitude:    0,
				Longitude:   0,
				RequestedBy: &types.Source{},
			},
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, tc.l.Validate())
		})
	}
}
