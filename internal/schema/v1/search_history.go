package schema

import "github.com/Decentr-net/cerberus/internal/schema/types"

const (
	maxSearchEngineLength = 20
	maxSearchQueryLength  = 2000
)

// SearchHistory is user's search history.
type SearchHistory struct {
	Timestamp

	Engine string `json:"engine"`
	Query  string `json:"query"`
}

// Type ...
func (SearchHistory) Type() types.Type {
	return types.PDVSearchHistoryType
}

// Validate ...
func (d SearchHistory) Validate() bool {
	if d.Engine == "" || d.Query == "" {
		return false
	}

	if len(d.Engine) > maxSearchEngineLength || len(d.Query) > maxSearchQueryLength {
		return false
	}

	return d.Timestamp.Validate()
}

// MarshalJSON ...
func (d SearchHistory) MarshalJSON() ([]byte, error) {
	return types.MarshalPDVData(d)
}
