package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/Decentr-net/cerberus/internal/schema/v1"
)

func TestPDV_UnmarshalJSON(t *testing.T) {
	data := `
{
    "version": "v1",
	"pdv": [
		{
			"type": "advertiserId",
			"advertiser": "decentr",
			"id": "12345qwert"
		},
		{
			"timestamp": "2021-05-11T11:05:18Z",
			"type": "cookie",
			"source": {
				"host": "https://decentr.xyz",
				"path": "/"
			},
            "name": "my cookie",
            "value": "some value",
            "domain": "*",
            "hostOnly": true,
            "path": "*",
            "secure": true,
            "sameSite": "None",
            "expirationDate": 1861920000
        },
		{
			"timestamp": "2021-05-11T11:05:18Z",
			"type": "location",
			"latitude": 37.24064741897542,
			"longitude": -115.81599314492902,
			"requestedBy": null
		},
        {
            "type": "profile",
            "firstName": "John",
            "lastName": "Dorian",
            "bio": "Just cool guy",
            "gender": "male",
            "avatar": "http://john.dorian/avatar.png",
            "birthday": "1993-01-20"
        },
		{
			"timestamp": "2021-05-11T11:05:18Z",
			"type": "searchHistory",
			"engine": "decentr",
			"query": "the best crypto"
		}
	]
}`
	var p PDVWrapper
	require.NoError(t, json.Unmarshal([]byte(data), &p))

	d, err := json.Marshal(p.pdv.(*v1.PDV))
	require.NoError(t, err)

	assert.JSONEq(t, data, string(d))
	assert.True(t, p.Validate())
}