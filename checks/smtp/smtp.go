package smtp

import (
	"context"
	"fmt"
	"net/smtp"
)

// New creates a new SMTP health check function.
func TestConnection(client *smtp.Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := client.Noop(); err != nil {
			return fmt.Errorf("failed to ping smt Server: %w", err)
		}

		return nil
	}
}
