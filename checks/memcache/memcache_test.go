package memcache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusUp(t *testing.T) {
	// Arrange
	addr := "localhost:11211"

	check := New(addr)

	// Act
	err := check()

	// Assert
	require.NoError(t, err)
}
