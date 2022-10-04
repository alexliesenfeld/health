package imap

import (
	"context"
	"fmt"

	"github.com/emersion/go-imap/client"
)

// New creates a new IMAP health check function.
func New(client *client.Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if client.State() < 1 {
			return fmt.Errorf("failed to ping imap Server")
		}

		return nil
	}
}
