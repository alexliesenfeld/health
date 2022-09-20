package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// New creates a new Postgres specific database health check function. It is database driver
// agnostic and hence expects you to provide an already established database connection.
func New(db *sql.DB) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("failed to ping postgres: %w", err)
		}

		rows, err := db.QueryContext(ctx, `select version()`)
		if err != nil {
			return fmt.Errorf("failed to run test query: %w", err)
		}

		if err = rows.Close(); err != nil {
			return fmt.Errorf("failed to close selected rows: %w", err)
		}

		return nil
	}
}
