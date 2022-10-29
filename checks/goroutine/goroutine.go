package goroutine

import (
	"context"
	"fmt"
	"runtime"
)

// New creates a new health check function for currently existing goroutines.
// Health check fails if the currently existing goroutines is greater than the threshold
func New(threshold uint) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		count := runtime.NumGoroutine()
		if uint(count) > threshold {
			return fmt.Errorf("current number of goroutines %d is higher than the threshold %d", count, threshold)
		}
		return nil
	}
}
