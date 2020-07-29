//+build swagger

package server

// swagger:model
type signature struct {
	// User's public key in hex.
	PublicKey string `json:"public_key"`
	// Request's digest signature in hex. Digest is sha256 hash of whole request data (except signature object).
	// You must hash data in order according to request attributes order.
	Signature string `json:"signature"`
}

// swagger:model
type sendPDVRequest struct {
	// Request signature
	Signature signature `json:"signature"`

	// Data which encrypted with base64
	Data string `json:"data"`
}

// swagger:model
type sendPDVResponse struct {
	// Put file address(hash in ipfs)
	Address string `json:"address"`
}

// swagger:model
type receivePDVRequest struct {
	// Request signature
	Signature signature `json:"signature"`

	// Requested file's address(hash in ipfs)
	Address string `json:"address"`
}

// swagger:model
type receivePDVResponse struct {
	// Requested file's data encrypted with base64
	Data string `json:"data"`
}

// DoesPDVExistRequest ...
type doesPDVExistRequest struct {
	// Request signature
	Signature signature `json:"signature"`

	// Requested file's address(hash in ipfs)
	Address string `json:"address"`
}

// swagger:model
type doesPDVExistResponse struct {
	// Flag which means file exists or not
	PDVExists bool `json:"pdv_exists"`
}

// swagger:model
type error struct {
	// error message
	Error string `json:"error"`
}
