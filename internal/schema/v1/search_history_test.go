package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSearchHistory_Validate(t *testing.T) {
	tt := []struct {
		name  string
		d     SearchHistory
		valid bool
	}{
		{
			name: "valid",
			d: SearchHistory{
				Timestamp: Timestamp{Time: time.Now()},
				Engine:    "decentr.xyz",
				Query:     "the best crypto",
			},
			valid: true,
		},
		{
			name: "empty engine",
			d: SearchHistory{
				Timestamp: Timestamp{Time: time.Now()},
				Engine:    "",
				Query:     "the best crypto",
			},
			valid: false,
		},
		{
			name: "empty searchLine",
			d: SearchHistory{
				Timestamp: Timestamp{Time: time.Now()},
				Engine:    "decentr.xyz",
				Query:     "",
			},
			valid: false,
		},
		{
			name: "invalid timestamp",
			d: SearchHistory{
				Engine: "decentr.xyz",
				Query:  "something",
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
