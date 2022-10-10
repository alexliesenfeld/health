package imap

import (
	"context"
	"log"
	"testing"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
	"github.com/stretchr/testify/require"
)

func TestStatusUp(t *testing.T) {

	//Create new memory imap backend
	go newImapServer()

	// Arrange
	check := New("localhost:1143")

	// Act
	err := check(context.Background())

	// Assert
	require.NoError(t, err)
}

func newImapServer() {
	// Create a memory backend
	be := memory.New()

	// Create a new server
	s := server.New(be)
	s.Addr = ":1143"
	// Since we will use this server for testing only, we can allow plain text
	// authentication over unencrypted connections
	s.AllowInsecureAuth = true

	//Close server afterwards
	defer s.Close()

	// log.Println("Starting IMAP server at localhost:1143")
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
