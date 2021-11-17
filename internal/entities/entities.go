// Package entities contains service-wide models.
package entities

import (
	"time"

	"github.com/Decentr-net/cerberus/internal/schema"
)

// PDVMeta contains info about PDV.
type PDVMeta struct {
	// ObjectTypes represents how much certain meta data meta contains.
	ObjectTypes map[schema.Type]uint16 `json:"object_types"`
	Reward      uint64                 `json:"reward"`
}

// Profile ...
type Profile struct {
	Address   string
	FirstName string
	LastName  string
	Emails    []string
	Bio       string
	Avatar    string
	Gender    string
	Birthday  time.Time
	UpdatedAt *time.Time
	CreatedAt time.Time
}
