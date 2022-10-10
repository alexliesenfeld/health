package imap

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/emersion/go-imap/client"
)

// New creates a new IMAP health check function.
func New(addr string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		c, err := client.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to imap server")
		}

		return check(ctx, c)
	}
}

// NewWithTLS creates a new IMAP health check function with enabled TLS security.
func NewWithTLS(addr string, config *tls.Config) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		c, err := client.DialTLS(addr, config)
		if err != nil {
			return fmt.Errorf("failed to connect to imap server")
		}

		return check(ctx, c)
	}
}

func check(_ context.Context, c *client.Client) error {
	state := c.State()
	if state < 1 {
		return fmt.Errorf("connection state check failure: expected connection status to be < 1 but was %v", state)
	}

	return nil
}
