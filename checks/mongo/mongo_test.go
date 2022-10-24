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
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI("mongodb://test:test@localhost:27017/?compressors=disabled&gssapiServiceName=mongod"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	check := New("mongodb://test:test@localhost:27017/?compressors=disabled&gssapiServiceName=mongod")

	// Act
	err = check(ctx)

	// Assert
	require.NoError(t, err)
}
