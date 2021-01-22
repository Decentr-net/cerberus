package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPDV_UnmarshalJSON(t *testing.T) {
	data := `
{
    "version": "v1",
	"pdv": [
		{
	        "domain": "decentr.net",
	        "path": "/",
		    "data": [
		        {
		            "version": "v1",
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
		            "version": "v1",
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
		},
		{
	        "domain": "mydomain.net",
	        "path": "/",
		    "data": [
		        {
		            "version": "v1",
		            "type": "cookie",
		            "name": "my cookie",
		            "value": "some value",
		            "domain": "*",
		            "host_only": true,
		            "path": "*",
		            "secure": true,
		            "same_site": "None",
		            "expiration_date": 1861920000
		        }
		    ]
		}
	]
}`

	var p PDV
	require.NoError(t, json.Unmarshal([]byte(data), &p))

	d, err := json.Marshal(p)
	require.NoError(t, err)

	require.JSONEq(t, data, string(d))
}
