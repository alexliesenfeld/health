package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// New creates a new MongoDB health check function.
// It is expects you to provide an already established MongoDB connection.
func New(client *mongo.Client, rp *readpref.ReadPref) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := client.Ping(ctx, rp); err != nil {
			return fmt.Errorf("failed to ping mongodb %w", err)
		}

		return nil
	}
}
