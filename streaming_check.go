package health

import (
	"context"
	"time"
)

func (c *streamingCheck) start(input AsyncCheckInput) chan error {
	return c.makeCheckStream(input)
}

func (c *BaseCheck) name() string {
	return c.Name
}

func (c *BaseCheck) interceptors() []Interceptor {
	return c.Interceptors
}

func (c *BaseCheck) onStateChange(ctx context.Context, oldState CheckState, newState CheckState) {
	c.StatusListener.onStateChange(ctx, c.Name, oldState, newState)
}

// maxFails implements evaluateCheckConfig.
func (c *BaseCheck) maxFails() uint {
	return c.MaxContiguousFails
}

// maxTimeInError implements evaluateCheckConfig.
func (c *BaseCheck) maxTimeInError() time.Duration {
	return c.MaxTimeInError
}
