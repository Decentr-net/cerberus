package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPDVType_UnmarshalText(t *testing.T) {
	var p Type
	require.NoError(t, p.UnmarshalText([]byte(PDVCookieType)))
	require.Error(t, p.UnmarshalText([]byte("wrong")))
}

func Test_MarshalPDVData(t *testing.T) {
	v := testPDVType{
		V: "string",
	}

	b, err := MarshalPDVData(v)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"v": "string",
		"type": "V"
	}`, string(b))
}

type testPDVType struct {
	V string `json:"v"`
}

func (testPDVType) Type() Type {
	return "V"
}

func (testPDVType) Validate() bool {
	return true
}
