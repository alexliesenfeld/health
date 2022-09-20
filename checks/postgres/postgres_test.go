package postgres

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStatusUp(t *testing.T) {
	// Arrange
	db, err := sql.Open("postgres", "postgres://test:test@localhost:5432/test?sslmode=disable")
	require.NoError(t, err)

	check := New(db)

	// Act
	err = check(context.Background())

	// Assert
	require.NoError(t, err)
}
