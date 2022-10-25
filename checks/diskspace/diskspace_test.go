package diskspace

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

// Set up with threshold of maximum value of uint64 and expect no failure
func TestExternalCheckWithMaximumThreshold(t *testing.T) {
	check := New(^uint64(0), "/")
	err := check(context.Background())
	require.NoError(t, err)
}

// Set up with threshold of zero and expect a failure
func TestExternalCheckWithMinimumThreshold(t *testing.T) {
	check := New(0, "/")
	err := check(context.Background())
	require.Error(t, err)
}

// Checks the "private" function that does value comparison with many test cases
func TestCheckInternalMethodOnManyInputs(t *testing.T) {
	// Need to test all possibilities for check function
	cases := []struct {
		Ctx           context.Context
		Threshold     uint64
		Total         uint64
		Available     uint64
		ErrorExpected bool
	}{
		{nil, 100, 200, 120, false},
		{nil, 100, 200, 180, false},
		{nil, 20, 200, 120, true},
		{nil, 0, 0, 0, false},
	}
	for _, tc := range cases {
		err := check(tc.Ctx, tc.Threshold, tc.Total, tc.Available)
		if tc.ErrorExpected {
			if err == nil {
				t.Errorf("Expected an error for {Threshold: %d, Total:%d, Available:%d}, did not recieve one.", tc.Threshold, tc.Total, tc.Available)
			}
		} else {
			if err != nil {
				t.Errorf("Did not expect an error, received %s", err)
			}
		}
	}
}
