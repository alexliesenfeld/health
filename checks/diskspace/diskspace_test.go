package diskspace

import (
	"context"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

// Set up with threshold of maximum value of uint64 and expect no failure
func TestExternalCheckWithMaximumThreshold(t *testing.T) {
	check := New(math.MaxUint64, "/")
	err := check(context.Background())
	require.NoError(t, err)
}

// Set up with threshold of zero and expect a failure
func TestExternalCheckWithMinimumThreshold(t *testing.T) {
	check := New(0, "/")
	err := check(context.Background())
	require.Error(t, err)
}
