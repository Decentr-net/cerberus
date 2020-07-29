package api

// Error ...
type Error struct {
	Error string `json:"error"`
}

// SendPDVResponse ...
type SendPDVResponse struct {
	Address string `json:"address"`
}

// ReceivePDVResponse ...
type ReceivePDVResponse struct {
	Data []byte `json:"data"`
}

// DoesPDVExistResponse ...
type DoesPDVExistResponse struct {
	PDVExists bool `json:"pdv_exists"`
}
