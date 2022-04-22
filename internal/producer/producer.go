// Package producer contains the interface of producer.
package producer

import (
	"context"

	"github.com/Decentr-net/cerberus/internal/entities"
)

//go:generate mockgen -destination=./mock/producer.go -package=mock -source=producer.go

// PDVMessage ...
type PDVMessage struct {
	ID      uint64
	Address string
	Device  string
	Meta    *entities.PDVMeta
	Data    []byte
}

// Producer ...
type Producer interface {
	Produce(ctx context.Context, m *PDVMessage) error
}
