// Package consumer contains interface of blocks consumer.
package consumer

import (
	"context"
)

//go:generate mockgen -destination=./mock/consumer.go -package=consumer -source=consumer.go

// Consumer consumes blocks from decentr blockchain.
type Consumer interface {
	Run(ctx context.Context) error
}
