package throttler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestThrottler_Reset(t *testing.T) {
	const key = "1"

	tr := New(1 * time.Second)

	require.False(t, tr.Throttle(key))
	tr.Reset(key)
	require.True(t, tr.Throttle(key))
	time.Sleep(2 * time.Second)
	require.False(t, tr.Throttle(key))
}
