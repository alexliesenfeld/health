package redis

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestStatusUp(t *testing.T) {
	// Arrange
	redis := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	check := New(redis)

	// Act
	err := check(context.Background())

	// Assert
	require.NoError(t, err)
}
