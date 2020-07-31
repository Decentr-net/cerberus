package api

import "regexp"

var addressRegExp = regexp.MustCompile(`[0-9a-fA-F]{76}\/[0-9a-fA-F]{128}`) // public_key_hex/data_sha256_digest_hex

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
	return IsAddressValid(r.Address)
}

// IsAddressValid check is address is matching with regexp.
func IsAddressValid(s string) bool {
	return addressRegExp.MatchString(s)
}
