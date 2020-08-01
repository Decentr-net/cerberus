package schema

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
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
        "ip": "1.1.1.1",
        "user_agent": "mac",
        "data": [
            {
                "version": "v1",
                "type": "cookie",
                "name": "my cookie",
                "value": "some value",
                "expires": "some date",
                "max_age": 1234,
                "path": "path",
                "domain": "domain"
            },
            {
                "version": "v1",
                "type": "cookie",
                "name": "my cookie",
                "value": "some value",
                "expires": "some date",
                "max_age": 1234,
                "path": "path",
                "domain": "domain"
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

			var re = regexp.MustCompile(`[ \n\t]`)
			require.Equal(t, re.ReplaceAllString(tc.data, ""), re.ReplaceAllString(string(d), ""))
		})
	}
}

func TestPDVObjectV1_Validate(t *testing.T) {
	tt := []struct {
		name  string
		o     PDVObjectV1
		valid bool
	}{
		{
			name: "valid ipv4",
			o: PDVObjectV1{
				IP:        "1.1.1.1",
				UserAgent: "user_agent",
				Data: []PDVData{
					&PDVDataCookieV1{
						Name:  "name",
						Value: "value",
					},
				},
			},
			valid: true,
		},
		{
			name: "valid ipv6",
			o: PDVObjectV1{
				IP:        "2001:0000:3238:DFE1:63:0000:0000:FEFB",
				UserAgent: "user_agent",
				Data: []PDVData{
					&PDVDataCookieV1{
						Name:  "name",
						Value: "value",
					},
				},
			},
			valid: true,
		},
		{
			name: "invalid ip",
			o: PDVObjectV1{
				IP:        "1a.1.1.1",
				UserAgent: "user_agent",
				Data: []PDVData{
					&PDVDataCookieV1{
						Name:  "name",
						Value: "value",
					},
				},
			},
			valid: false,
		},
		{
			name: "empty ua",
			o: PDVObjectV1{
				IP: "1.1.1.1",
				Data: []PDVData{
					&PDVDataCookieV1{
						Name:  "name",
						Value: "value",
					},
				},
			},
			valid: false,
		},
		{
			name: "empty data",
			o: PDVObjectV1{
				IP:   "1.1.1.1",
				Data: []PDVData{},
			},
			valid: false,
		},
		{
			name: "invalid data",
			o: PDVObjectV1{
				IP:        "1.1.1.1",
				UserAgent: "user_agent",
				Data: []PDVData{
					&PDVDataCookieV1{},
				},
			},
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, tc.o.Validate())
		})
	}
}

func TestPDVDataCookieV1_Validate(t *testing.T) {
	tt := []struct {
		name  string
		c     PDVDataCookieV1
		valid bool
	}{
		{
			name: "valid",
			c: PDVDataCookieV1{
				Name:    "name",
				Value:   "valie",
				Expires: "expires",
				MaxAge:  1,
				Path:    "p",
				Domain:  "d",
			},
			valid: true,
		},
		{
			name: "valid minimal",
			c: PDVDataCookieV1{
				Name:  "name",
				Value: "valie",
			},
			valid: true,
		},
		{
			name: "without name",
			c: PDVDataCookieV1{
				Value: "value",
			},
			valid: false,
		},
		{
			name: "without value",
			c: PDVDataCookieV1{
				Name: "name",
			},
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, tc.c.Validate())
		})
	}
}
