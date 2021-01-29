package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPDV_Validate(t *testing.T) {
	o := PDVObjectV1{
		PDVObjectMetaV1: PDVObjectMetaV1{
			Host: "decentr.net",
			Path: "path",
		},
		Data: []PDVData{
			&PDVDataCookie{
				Name:  "name",
				Value: "value",
			},
		},
	}

	tt := []struct {
		name  string
		p     PDV
		valid bool
	}{
		{
			name: "valid",
			p: PDV{
				Version: PDVV1,
				PDV:     []PDVObject{&o},
			},
			valid: true,
		},
		{
			name: "wrong_version",
			p: PDV{
				Version: "wrong",
				PDV:     []PDVObject{&o},
			},
			valid: false,
		},
		{
			name: "empty_data",
			p: PDV{
				Version: PDVV1,
				PDV:     []PDVObject{},
			},
			valid: false,
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.valid, tc.p.Validate())
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
			name: "valid",
			o: PDVObjectV1{
				PDVObjectMetaV1: PDVObjectMetaV1{
					Host: "decentr.net",
					Path: "path",
				},
				Data: []PDVData{
					&PDVDataCookie{
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
					&PDVDataCookie{
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
					&PDVDataCookie{
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
					&PDVDataCookie{
						Name:  "name",
						Value: "value",
					},
				},
			},
			valid: true,
		},
		{
			name: "empty Data",
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
					&PDVDataCookie{
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
					&PDVDataCookie{
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
					&PDVDataCookie{
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
					&PDVDataCookie{
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
