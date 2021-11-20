// Package producer contains the interface of producer.
package producer

import (
	"context"
	"encoding/json"

	"github.com/Decentr-net/cerberus/internal/entities"
)

//go:generate mockgen -destination=./mock/producer.go -package=mock -source=producer.go

// PDVMessage ...
type PDVMessage struct {
	ID      uint64
	Address string
	Meta    *entities.PDVMeta
	Data    json.RawMessage
}

// Producer ...
type Producer interface {
	Produce(ctx context.Context, m *PDVMessage) error
}
