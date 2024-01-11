package health

import (
	"context"
	"fmt"
	"sync"

	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusUnknownBeforeStatusUp(t *testing.T) {
	// Arrange
	testData := map[string]CheckState{"check1": {Status: StatusUp}, "check2": {Status: StatusUnknown}}

	// Act
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, result, StatusUnknown)
}

func TestStatusDownBeforeStatusUnknown(t *testing.T) {
	// Arrange
	testData := map[string]CheckState{"check1": {Status: StatusDown}, "check2": {Status: StatusUnknown}}

	// Act
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, result, StatusDown)
}

func doTestEvaluateAvailabilityStatus(
	t *testing.T,
	expectedStatus AvailabilityStatus,
	maxTimeInError time.Duration,
	maxFails uint,
	state CheckState,
) {
	// Act
	result := evaluateCheckStatus(&state, mockEvaluateCheckConfig{maxTimeInError, maxFails})

	// Assert
	assert.Equal(t, expectedStatus, result)
}

type mockEvaluateCheckConfig struct {
	MaxTimeInError time.Duration
	MaxFails       uint
}

// maxFails implements evaluateCheckConfig.
func (c mockEvaluateCheckConfig) maxFails() uint {
	return c.MaxFails
}

// maxTimeInError implements evaluateCheckConfig.
func (c mockEvaluateCheckConfig) maxTimeInError() time.Duration {
	return c.MaxTimeInError
}

func TestWhenNoChecksMadeYetThenStatusUnknown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUnknown, 0, 0, CheckState{
		LastCheckedAt: time.Time{},
	})
}

func TestWhenNoErrorThenStatusUp(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 0, CheckState{
		LastCheckedAt: time.Now(),
	})
}

func TestWhenErrorThenStatusDown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusDown, 0, 0, CheckState{
		LastCheckedAt: time.Now(),
		Result:        fmt.Errorf("example error"),
	})
}

func TestWhenErrorAndMaxFailuresThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Second, uint(10), CheckState{
		LastCheckedAt:       time.Now(),
		Result:              fmt.Errorf("example error"),
		FirstCheckStartedAt: time.Now().Add(-2 * time.Minute),
		LastSuccessAt:       time.Now().Add(-3 * time.Minute),
		ContiguousFails:     1,
	})
}

func TestWhenErrorAndMaxTimeInErrorThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Hour, uint(1), CheckState{
		LastCheckedAt:       time.Now(),
		Result:              fmt.Errorf("example error"),
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
		LastSuccessAt:       time.Now().Add(-2 * time.Minute),
		ContiguousFails:     100,
	})
}

func TestWhenErrorAndAllThresholdsCrossedThenStatusDown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusDown, 1*time.Second, uint(1), CheckState{
		LastCheckedAt:       time.Now(),
		Result:              fmt.Errorf("example error"),
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
		LastSuccessAt:       time.Now().Add(-2 * time.Minute),
		ContiguousFails:     5,
	})
}

func TestStartStopManualPeriodicChecks(t *testing.T) {
	ckr := NewChecker(
		WithDisabledAutostart(),
		WithPeriodicCheck(50*time.Minute, 0, Check{
			Name: "check",
			Check: func(ctx context.Context) error {
				return nil
			},
		}))

	assert.Equal(t, 0, ckr.GetRunningPeriodicCheckCount())

	ckr.Start()
	assert.Equal(t, 1, ckr.GetRunningPeriodicCheckCount())

	ckr.Stop()
	assert.Equal(t, 0, ckr.GetRunningPeriodicCheckCount())
}

func doTestCheckerCheckFunc(t *testing.T, updateInterval time.Duration, err error, expectedStatus AvailabilityStatus) {
	// Arrange
	streamingCheckChange := make(chan struct{})
	onStreamingCheckChange := sync.OnceFunc(func() {
		close(streamingCheckChange)
	})

	ckr := NewChecker(
		WithTimeout(10*time.Second),
		WithCheck(Check{
			Name: "check1",
			Check: func(ctx context.Context) error {
				return nil
			},
		}),
		WithPeriodicCheck(updateInterval, 0, Check{
			Name: "check2",
			Check: func(ctx context.Context) error {
				return err
			},
		}),
		WithStreamingCheck(StreamingCheck{
			Name: "check3",
			StatusListener: func(ctx context.Context, name string, state CheckState) {
				onStreamingCheckChange()
			},
			MakeCheckStream: func(ctx context.Context) chan error {
				checkStream := make(chan error)
				go func() {
					defer close(checkStream)
					for {
						checkStream <- nil
						select {
						case <-time.After(updateInterval):
							continue
						case <-ctx.Done():
							checkStream <- ctx.Err()
							return
						}
					}
				}()
				return checkStream
			},
		}),
	)

	if updateInterval == 0 {
		// If the updateInterval is 0,
		// wait for the streaming check to publish it's first status change
		// so that the correct status (not Unknown) is received from the next Check call.
		<-streamingCheckChange
		// This is not necessary for the periodic check,
		// since a periodic check with an update interval of 0
		// is treated exactly like a synchronous check.
	}

	// Act
	res := ckr.Check(context.Background())

	// Assert
	require.NotNil(t, res.Details)
	assert.Equal(t, expectedStatus, res.Status)
	for _, checkName := range []string{"check1", "check2", "check3"} {
		_, checkResultExists := res.Details[checkName]
		assert.True(t, checkResultExists)
	}
}

func TestWhenChecksExecutedThenAggregatedResultUp(t *testing.T) {
	doTestCheckerCheckFunc(t, 0, nil, StatusUp)
}

func TestWhenOneCheckFailedThenAggregatedResultDown(t *testing.T) {
	doTestCheckerCheckFunc(t, 0, fmt.Errorf("this is a check error"), StatusDown)
}

func TestCheckSuccessNotAllChecksExecutedYet(t *testing.T) {
	doTestCheckerCheckFunc(t, 5*time.Hour, nil, StatusUnknown)
}

func TestPanicRecovery(t *testing.T) {
	// Arrange
	expectedPanicMsg := "test message"
	ckr := NewChecker(
		WithCheck(Check{
			Name: "iPanic",
			Check: func(ctx context.Context) error {
				panic(expectedPanicMsg)
			},
		}),
	)

	// Act
	res := ckr.Check(context.Background())

	// Assert
	require.NotNil(t, res.Details)
	assert.Equal(t, StatusDown, res.Status)

	checkRes, checkResultExists := res.Details["iPanic"]
	assert.True(t, checkResultExists)
	assert.NotNil(t, checkRes.Error)
	assert.Equal(t, (checkRes.Error).Error(), expectedPanicMsg)
}
