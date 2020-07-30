package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendPDVRequest_IsValid(t *testing.T) {
	assert.True(t, SendPDVRequest{Data: []byte{1}}.IsValid())
	assert.False(t, SendPDVRequest{}.IsValid())
}

func TestReceivePDVRequest_IsValid(t *testing.T) {
	assert.True(t, ReceivePDVRequest{Address: testAddress}.IsValid())
	assert.False(t, ReceivePDVRequest{}.IsValid())
}

func TestIsAddressValid(t *testing.T) {
	tt := []struct {
		name    string
		address string
		valid   bool
	}{
		{
			name:    "valid",
			address: "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3a",
			valid:   true,
		},
		{
			name:    "not_hex",
			address: "zb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3a",
			valid:   false,
		},
		{
			name:    "missed_digest",
			address: "zb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/",
			valid:   false,
		},
		{
			name:    "invalid_pk",
			address: "b5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3a",
			valid:   false,
		},
		{
			name:    "invalid_digest",
			address: "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3",
			valid:   false,
		},
		{
			name:    "invalid_empty",
			address: "eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3ae2fc6e298ed6/eb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3aeb5ae98721035133ec05dfe1052ddf78f57dc4b018cedb0c2726261d165dad3",
			valid:   false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, IsAddressValid(tc.address))
		})
	}
}
