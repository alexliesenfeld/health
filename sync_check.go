package health

import (
	"context"
	"time"
)

func (c *Check) onStateChange(ctx context.Context, oldState CheckState, newState CheckState) {
	c.StatusListener.onStateChange(ctx, c.Name, oldState, newState)
}

// maxFails implements evaluateCheckConfig.
func (c *Check) maxFails() uint {
	return c.MaxContiguousFails
}

// maxTimeInError implements evaluateCheckConfig.
func (c *Check) maxTimeInError() time.Duration {
	return c.MaxTimeInError
}
