// Package throttler provides functionality to throttle http requests
package throttler

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// Throttler ...
type Throttler interface {
	Throttle(key string) bool
	Reset(key string)
}

type throttler struct {
	c *cache.Cache
}

// New returns a new instance of Throttler.
func New(period time.Duration) Throttler {
	return &throttler{
		c: cache.New(period, time.Hour),
	}
}

// Throttle ...
func (t *throttler) Throttle(key string) bool {
	_, ok := t.c.Get(key)
	return ok
}

// Reset ...
func (t *throttler) Reset(key string) {
	t.c.SetDefault(key, true)
}
