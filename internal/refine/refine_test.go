package refine

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Decentr-net/cerberus/pkg/schema"
)

func TestCookie(t *testing.T) {
	tt := []struct {
		str   string
		valid bool
	}{
		{"aaaa", true},
		{"aaaaaa", false},
		{"zzzzzzzzz", false},
		{"123456", true},
		{"1234567890", true},
		{"русский", true},
		{"ђђђђђђђђ", false},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.str, func(t *testing.T) {
			require.Equal(t, tc.valid, Cookie(&schema.V1Cookie{Value: tc.str}))
		})
	}
}

func TestSearchHistory(t *testing.T) {
	tt := []struct {
		str   string
		valid bool
	}{
		{"aaaa", true},
		{"aaaaaa", false},
		{"123456", true},
		{"1234567890", true},
		{"русский", true},
		{"ђђђђђђђђ", false},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.str, func(t *testing.T) {
			require.Equal(t, tc.valid, SearchHistory(&schema.V1SearchHistory{Query: tc.str}))
		})
	}
}
