package schema

import (
	"github.com/Decentr-net/cerberus/pkg/schema/types"
)

const (
	maxSearchEngineLength = 20
	maxDomainLength       = 255
	maxSearchQueryLength  = 2000
)

// SearchHistory is user's search history.
type SearchHistory struct {
	types.Timestamp

	Domain string `json:"domain"`
	Engine string `json:"engine"`
	Query  string `json:"query"`
}

// Type ...
func (SearchHistory) Type() types.Type {
	return types.PDVSearchHistoryType
}

// Validate ...
func (d SearchHistory) Validate() bool {
	if d.Engine == "" || d.Query == "" || d.Domain == "" {
		return false
	}

	if len(d.Engine) > maxSearchEngineLength ||
		len(d.Query) > maxSearchQueryLength ||
		len(d.Domain) > maxDomainLength {
		return false
	}

	return d.Timestamp.Validate()
}

// MarshalJSON ...
func (d SearchHistory) MarshalJSON() ([]byte, error) {
	return types.MarshalPDVData(d)
}
