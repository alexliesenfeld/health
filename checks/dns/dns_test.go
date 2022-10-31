package dns

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDNSHealthCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cases := []struct {
		host        string
		errExpected bool
	}{
		{"localhost", false},
		{"google.com", false},
		{"1.1.1.1", false},
		{"invalid.invalid", true}, // RFC 2606 reserves .invalid
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("DNS lookup for %s", tt.host), func(t *testing.T) {
			check := New(tt.host)
			err := check(ctx)
			if tt.errExpected && err == nil {
				t.Errorf("Expected error for DNS look of %s", tt.host)
			}
			if !tt.errExpected && err != nil {
				t.Errorf("Expected successful DNS lookup for host %s. Got error: %v", tt.host, err)
			}
		})
	}
}
