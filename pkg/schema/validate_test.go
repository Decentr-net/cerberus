package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPDVObjectV1_Validate(t *testing.T) {
	tt := []struct {
		name  string
		o     PDVObjectV1
		valid bool
	}{
		{
			name: "valid",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "decentr.net",
					Path: "path",
				},
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
			name: "valid 2",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "decentr.net",
					Path: "/path",
				},
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
			name: "valid 3",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "decentr.net",
					Path: "",
				},
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
			name: "valid 4",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "107.180.50.186",
					Path: "",
				},
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
			name: "empty data",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "decentr.net",
					Path: "path",
				},
				Data: []PDVData{},
			},
			valid: false,
		},
		{
			name: "invalid host",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "",
					Path: "path",
				},
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
			name: "invalid host 2",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "host",
					Path: "path",
				},
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
			name: "invalid host 3",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "1.1.1.1.1",
					Path: "path",
				},
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
			name: "invalid path",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "decentr.net",
					Path: string([]byte{0x7f}),
				},
				Data: []PDVData{
					&PDVDataCookieV1{
						Name:  "name",
						Value: "value",
					},
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
				Name:           "name",
				Value:          "value",
				Domain:         "decentr.net",
				HostOnly:       true,
				Path:           "*",
				Secure:         true,
				HTTPOnly:       true,
				SameSite:       "*",
				Session:        false,
				ExpirationDate: 123413412,
			},
			valid: true,
		},
		{
			name: "valid minimal",
			c: PDVDataCookieV1{
				Name:  "name",
				Value: "value",
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
