package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestStatusUp(t *testing.T) {
	// Arrange
	ctx := context.Background()
	addr := "mongodb://test:test@localhost:27017/?compressors=disabled&gssapiServiceName=mongod"
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI(addr))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	check := New(addr)

	// Act
	err = check(ctx)

	// Assert
	require.NoError(t, err)
}
