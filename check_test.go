package health

import (
	"context"
	"fmt"
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
	maxTimeInError,
	minTimeInSuccess time.Duration,
	maxFails,
	minSuccess uint,
	state CheckState,
) {
	// Arrange
	check := &Check{
		MaxTimeInError:         maxTimeInError,
		MinTimeInSuccess:       minTimeInSuccess,
		MaxContiguousFails:     maxFails,
		MinContiguousSuccesses: minSuccess,
	}

	// Act
	result := evaluateCheckStatus(&state, check)

	// Assert
	assert.Equal(t, expectedStatus, result)
}

func TestWhenNoChecksMadeYetThenStatusUnknown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUnknown, 0, 0, 0, 0, CheckState{
		LastCheckedAt: &time.Time{},
	})
}

func TestWhenNoErrorThenStatusUp(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 0, 0, 0, CheckState{
		LastCheckedAt: &now,
	})
}

func TestWhenErrorThenStatusDown(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusDown, 0, 0, 0, 0, CheckState{
		LastCheckedAt: &now,
		Result:        fmt.Errorf("example error"),
	})
}

func TestWhenErrorAndMaxFailuresThresholdNotCrossedThenStatusWarn(t *testing.T) {
	now := time.Now()
	lastSuccessAt := now.Add(-3 * time.Minute)

	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Second, 0, uint(10), 0, CheckState{
		LastCheckedAt:       &now,
		Result:              fmt.Errorf("example error"),
		FirstCheckStartedAt: now.Add(-2 * time.Minute),
		LastSuccessAt:       &lastSuccessAt,
		ContiguousFails:     1,
	})
}

func TestWhenErrorAndMaxTimeInErrorThresholdNotCrossedThenStatusWarn(t *testing.T) {
	now := time.Now()
	lastSuccessAt := now.Add(-2 * time.Minute)
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Hour, 0, uint(1), 0, CheckState{
		LastCheckedAt:       &now,
		Result:              fmt.Errorf("example error"),
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
		LastSuccessAt:       &lastSuccessAt,
		ContiguousFails:     100,
	})
}

func TestWhenErrorAndAllThresholdsCrossedThenStatusDown(t *testing.T) {
	now := time.Now()
	lastSuccessAt := now.Add(-2 * time.Minute)
	doTestEvaluateAvailabilityStatus(t, StatusDown, 1*time.Second, 0, uint(1), 0, CheckState{
		LastCheckedAt:       &now,
		Result:              fmt.Errorf("example error"),
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
		LastSuccessAt:       &lastSuccessAt,
		ContiguousFails:     5,
	})
}

func TestWhenErrorAtStartupBelowThresholdThenStatusDown(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusDown, 0, 0, uint(5), 0, CheckState{
		LastCheckedAt:   &now,
		Result:          fmt.Errorf("example error"),
		ContiguousFails: 1,
	})
}

func TestWhenSuccessAndMinSuccessThresholdNotCrossedThenUnknown(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusUnknown, 0, 0, 0, uint(10), CheckState{
		Status:              StatusUnknown,
		LastCheckedAt:       &now,
		ContiguousSuccesses: 5,
	})
}

func TestWhenSuccessAndMinTimeSinceStartedNotCrossedThenUnknown(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusUnknown, 0, 10*time.Second, 0, 0, CheckState{
		Status:              StatusUnknown,
		LastCheckedAt:       &now,
		FirstCheckStartedAt: time.Now().Add(-5 * time.Second),
	})
}

func TestWhenSuccessAndMinTimeSinceStartedCrossedThenUp(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 10*time.Second, 0, 0, CheckState{
		Status:              StatusUnknown,
		LastCheckedAt:       &now,
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
	})
}

func TestWhenSuccessAndMinSuccessTimeNotCrossedThenDown(t *testing.T) {
	now := time.Now()
	lastFailureAt := time.Now().Add(-5 * time.Second)
	doTestEvaluateAvailabilityStatus(t, StatusDown, 0, 10*time.Second, 0, 0, CheckState{
		Status:              StatusDown,
		LastCheckedAt:       &now,
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
		LastFailureAt:       &lastFailureAt,
	})
}

func TestWhenSuccessAndMinSuccessTimeCrossedThenUp(t *testing.T) {
	now := time.Now()
	lastFailureAt := time.Now().Add(-3 * time.Minute)
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 10*time.Second, 0, 0, CheckState{
		Status:              StatusDown,
		LastCheckedAt:       &now,
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
		LastFailureAt:       &lastFailureAt,
	})
}

func TestWhenSuccessAndMinSuccessThresholdCrossedThenUp(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 0, 0, uint(5), CheckState{
		Status:              StatusUnknown,
		LastCheckedAt:       &now,
		ContiguousSuccesses: 10,
	})
}

func TestWhenSuccessAndMinSuccessThresholdAndTimeCrossedThenUp(t *testing.T) {
	now := time.Now()
	lastFailureAt := time.Now().Add(-3 * time.Minute)
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 10*time.Second, 0, uint(5), CheckState{
		Status:              StatusDown,
		LastCheckedAt:       &now,
		ContiguousSuccesses: 10,
		LastFailureAt:       &lastFailureAt,
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
	)

	// Act
	res := ckr.Check(context.Background())

	// Assert
	require.NotNil(t, res.Details)
	assert.Equal(t, expectedStatus, res.Status)
	for _, checkName := range []string{"check1", "check2"} {
		_, checkResultExists := (*res.Details)[checkName]
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
