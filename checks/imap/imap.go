package imap

import (
	"context"
	"fmt"

	"github.com/emersion/go-imap/client"
)

// New creates a new IMAP health check function.
func New(addr string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		client, err := client.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to ping imap server")
		}
		if client.State() < 1 {
			return fmt.Errorf("failed to validate connection state to server")
		}

		return nil
	}
}
