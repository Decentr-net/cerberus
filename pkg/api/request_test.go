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
	assert.True(t, ReceivePDVRequest{Address: "fdsfsdfs"}.IsValid())
	assert.False(t, ReceivePDVRequest{}.IsValid())
}

func TestDoesPDVExistRequest_IsValid(t *testing.T) {
	assert.True(t, DoesPDVExistRequest{Address: "fdsfsdfs"}.IsValid())
	assert.False(t, DoesPDVExistRequest{}.IsValid())
}
