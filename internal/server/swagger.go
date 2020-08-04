//+build swagger

package server

// swagger:model SavePDVResponse
type savePDVResponse struct {
	// Put file address
	Address string `json:"address"`
}

// swagger:model Error
type error struct {
	// error message
	Error string `json:"error"`
}
