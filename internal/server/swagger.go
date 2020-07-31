//+build swagger

package server

// swagger:model SendPDVResponse
type sendPDVResponse struct {
	// Put file address
	Address string `json:"address"`
}

// swagger:model Error
type error struct {
	// error message
	Error string `json:"error"`
}
