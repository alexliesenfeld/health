package dns

import (
	"context"
	"net"
)

// New creates a health check function for DNS resolution of the specified host
func New(host string) func(ctx context.Context) error {
	resolver := new(net.Resolver)
	return func(ctx context.Context) error {
		_, err := resolver.LookupHost(ctx, host)
		return err
	}
}
