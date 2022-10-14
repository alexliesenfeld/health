package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func TestStatusUp(t *testing.T) {
	// Arrange
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI("mongodb://test:test@localhost:27017/?compressors=disabled&gssapiServiceName=mongod"))
	require.NoError(t, err)

	check := New(client, readpref.Primary())

	// Act
	err = check(ctx)

	// Assert
	require.NoError(t, err)
}
