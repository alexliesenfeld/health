package memcache

import (
	"fmt"

	"github.com/bradfitz/gomemcache/memcache"
)

// New creates a new Memcache client health check function.
// It is expects you to provide an already established memcache connection.
func New(addr string) func() error {
	return func() error {
		client := memcache.New(addr)
		if err := client.Ping(); err != nil {
			return fmt.Errorf("failed to ping memcache: %w", err)
		}

		return nil
	}
}
