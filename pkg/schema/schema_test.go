package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDV_UnmarshalJSON(t *testing.T) {
	data := `
{
    "version": "v1",
	"device": "ios",
	"pdv": [
		{
			"type": "advertiserId",
			"advertiser": "decentr",
			"name": "12345qwert",
			"value": "12345value"
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
            "emails": ["dev@decentr.xyz"],
            "bio": "Just cool guy",
            "gender": "male",
            "avatar": "http://john.dorian/avatar.png",
            "birthday": "1993-01-20"
        },
		{
			"timestamp": "2021-05-11T11:05:18Z",
			"type": "searchHistory",
			"engine": "decentr",
			"domain": "decentr.xyz",
			"query": "the best crypto"
		}
	]
}`
	var p PDVWrapper
	require.NoError(t, json.Unmarshal([]byte(data), &p))

	d, err := json.Marshal(p)
	require.NoError(t, err)

	assert.JSONEq(t, data, string(d))
	assert.True(t, p.Validate())
}

func Test_GetInvalidPDV(t *testing.T) {
	s, err := GetInvalidPDV([]byte(`
{
   "version":"v1",
   "device": "ios",
   "pdv":[
      {
         "domain":".xn--j1ail.xn--p1ai",
         "expirationDate":1708541760,
         "hostOnly":false,
         "name":"__utma",
         "path":"/",
         "sameSite":"unspecified",
         "secure":false,
         "source":{
            "host":".xn--j1ail.xn--p1ai",
            "path":"/"
         },
         "timestamp":"2022-02-21T18:56:00.444Z",
         "type":"cookie",
         "value":"127263044.1244664020.1645459224.1645459224.1645469307.2"
      },
      {
         "domain":"t.tilda.ws",
         "expirationDate":1708541625,
         "hostOnly":false,
         "name":"_ga",
         "path":"/",
         "sameSite":"unspecified",
         "secure":false,
         "source":{
            "host":"t.tilda.ws",
            "path":"/"
         },
         "timestamp":"2022-02-21T18:53:45.982Z",
         "type":"cookie",
         "value":"GA1.2.214209808.1645469626"
      }
   ]
}`))
	require.NoError(t, err)
	require.Equal(t, s, []int{0})
}
