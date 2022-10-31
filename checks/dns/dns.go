package dns

import (
	"context"
	"fmt"
	"net"
)

// New creates a health check function for DNS resolution of the specified host
func New(host string) func(ctx context.Context) error {
	resolver := new(net.Resolver)
	return func(ctx context.Context) error {
		addrs, err := resolver.LookupHost(ctx, host)
		if err != nil {
			return err
		}
		if len(addrs) == 0 {
			return fmt.Errorf("could not resolve host")
		}
		return nil
	}
}
