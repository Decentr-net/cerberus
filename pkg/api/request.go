package api

// Validator interface provides method for validation.
type Validator interface {
	IsValid() bool
}

// SendPDVRequest ...
type SendPDVRequest struct {
	AuthRequest

	Data []byte `json:"data"`
}

// IsValid ...
func (r SendPDVRequest) IsValid() bool {
	return len(r.Data) != 0
}

// ReceivePDVRequest ...
type ReceivePDVRequest struct {
	AuthRequest

	Address string `json:"address"`
}

// IsValid ...
func (r ReceivePDVRequest) IsValid() bool {
	return len(r.Address) != 0
}

// DoesPDVExistRequest ...
type DoesPDVExistRequest struct {
	AuthRequest

	Address string `json:"address"`
}

// IsValid ...
func (r DoesPDVExistRequest) IsValid() bool {
	return len(r.Address) != 0
}
