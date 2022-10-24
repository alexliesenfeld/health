package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// New creates a new MongoDB health check function.
// It is expects you to provide an already established MongoDB connection.
func New(addr string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		client, err := mongo.Connect(ctx, options.Client().
			ApplyURI("mongodb://test:test@localhost:27017/?compressors=disabled&gssapiServiceName=mongod"))

		if err != nil {
			return err
		}

		if err := client.Ping(ctx, readpref.Primary()); err != nil {
			return fmt.Errorf("failed to ping mongodb %w", err)
		}

		err = client.Disconnect(ctx)

		if err != nil {
			return err
		}

		return nil
	}
}
