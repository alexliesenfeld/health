package redis

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
)

// New creates a new Redis health check function.
func New(client *redis.Client) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		pingResult := client.Ping(ctx)
		_, pingResultErr := pingResult.Result()
		if pingResultErr != nil {
			return errors.New("failed to connect to redis server: " + pingResult.String())
		}

		return nil
	}
}
