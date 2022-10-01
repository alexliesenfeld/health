package smtp

import (
	"context"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusUp(t *testing.T) {
	// Arrange
	client, err := smtp.Dial("localhost:25")
	require.NoError(t, err)

	check := TestConnection(client)

	// Act
	err = check(context.Background())

	// Assert
	require.NoError(t, err)
}
