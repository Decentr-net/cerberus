package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/Decentr-net/cerberus/internal/schema/v1"
)

func TestPDV_UnmarshalJSON(t *testing.T) {
	data := `
{
    "version": "v1",
	"pdv": [
		{
			"source": {
				"host": "https://decentr.xyz",
				"path": "/"
			},
            "type": "cookie",
            "name": "my cookie",
            "value": "some value",
            "domain": "*",
            "host_only": true,
            "path": "*",
            "secure": true,
            "same_site": "None",
            "expiration_date": 1861920000
        },
        {
			"source": {
				"host": "https://decentr.xyz",
				"path": "/login"
			},
            "type": "login_cookie",
            "name": "my cookie 2",
            "value": "some value 2",
            "domain": "*",
            "host_only": true,
            "path": "*",
            "secure": true,
            "same_site": "None",
            "expiration_date": 1861920000
        }
	]
}`

	var p PDVWrapper
	require.NoError(t, json.Unmarshal([]byte(data), &p))

	d, err := json.Marshal(p.pdv.(*v1.PDV))
	require.NoError(t, err)

	require.JSONEq(t, data, string(d))
}
