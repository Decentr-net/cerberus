package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPDV_UnmarshalJSON(t *testing.T) {
	tt := []struct {
		name        string
		data        string
		unmarshaled bool
	}{
		{
			name: "cookie",
			data: `{
    "version": "v1",
	"pdv": {
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
	            "http_only": true,
	            "same_site": "None",
	            "session": false,
	            "expiration_date": 1861920000
	        },
	        {
	            "version": "v1",
	            "type": "cookie",
	            "name": "my cookie 2",
	            "value": "some value 2",
	            "domain": "*",
	            "host_only": true,
	            "path": "*",
	            "secure": true,
	            "http_only": true,
	            "same_site": "None",
	            "session": false,
	            "expiration_date": 1861920000
	        }
	    ]
	}
}`,
			unmarshaled: true,
		},
	}

	for i := range tt {
		tc := tt[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var p PDV
			require.Equal(t, tc.unmarshaled, json.Unmarshal([]byte(tc.data), &p) == nil)

			d, err := json.Marshal(p)
			require.NoError(t, err)

			require.JSONEq(t, tc.data, string(d))
		})
	}
}
