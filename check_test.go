package health

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAggregateResult(t *testing.T) {
	// Arrange
	errMsg := "this is an error message"
	testData := map[string]CheckResult{
		"check1": {
			Status:    StatusUp,
			Error:     nil,
			Timestamp: time.Now().Add(-5 * time.Minute),
		},
		"check2": {
			Status:    StatusUnknown,
			Error:     nil,
			Timestamp: time.Now().Add(-3 * time.Minute),
		},
		"check3": {
			Status:    StatusDown,
			Error:     &errMsg,
			Timestamp: time.Now().Add(-1 * time.Minute),
		},
	}

	// Act
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, StatusDown, result)
	//assert.Equal(t, true, result.Timestamp.Equal(testData["check3"].Timestamp))
	//assert.Equal(t, true, reflect.DeepEqual(&testData, result.Details))
}

func TestAggregateResultWithoutDetails(t *testing.T) {
	// Arrange
	testData := map[string]CheckResult{"check1": {Status: StatusUp, Timestamp: time.Now()}}

	// Act
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, StatusUp, result)
}

func doTestEvaluateAvailabilityStatus(
	t *testing.T,
	expectedStatus Status,
	maxTimeInError time.Duration,
	maxFails uint,
	state checkState,
) {
	// Act
	result := evaluateAvailabilityStatus(&state, maxTimeInError, maxFails)

	// Assert
	assert.Equal(t, expectedStatus, result)
}

func TestWhenNoChecksMadeYetThenStatusUnknown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUnknown, 0, 0, checkState{
		lastCheckedAt: time.Time{},
	})
}

func TestWhenNoErrorThenStatusUp(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 0, checkState{
		lastCheckedAt: time.Now(),
	})
}

func TestWhenErrorThenStatusDown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusDown, 0, 0, checkState{
		lastCheckedAt: time.Now(),
		lastResult:    fmt.Errorf("example error"),
	})
}

func TestWhenErrorAndMaxFailuresThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Second, uint(10), checkState{
		lastCheckedAt:    time.Now(),
		lastResult:       fmt.Errorf("example error"),
		startedAt:        time.Now().Add(-3 * time.Minute),
		lastSuccessAt:    time.Now().Add(-2 * time.Minute),
		consecutiveFails: 1,
	})
}

func TestWhenErrorAndMaxTimeInErrorThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Hour, uint(1), checkState{
		lastCheckedAt:    time.Now(),
		lastResult:       fmt.Errorf("example error"), // Required for the test
		startedAt:        time.Now().Add(-3 * time.Minute),
		lastSuccessAt:    time.Now().Add(-2 * time.Minute),
		consecutiveFails: 100,
	})
}

func TestWhenErrorAndAllThresholdsCrossedThenStatusDown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusDown, 1*time.Second, uint(1), checkState{
		lastCheckedAt:    time.Now(),
		lastResult:       fmt.Errorf("example error"), // Required for the test
		startedAt:        time.Now().Add(-3 * time.Minute),
		lastSuccessAt:    time.Now().Add(-2 * time.Minute),
		consecutiveFails: 5,
	})
}

func TestToErrorDescErrorShortened(t *testing.T) {
	assert.Equal(t, "this", *toErrorDesc(fmt.Errorf("this is nice"), 4))
}

func TestToErrorDescErrorNotShortened(t *testing.T) {
	assert.Equal(t, "this is nice", *toErrorDesc(fmt.Errorf("this is nice"), 400))
}

func TestToErrorDescNoError(t *testing.T) {
	assert.Nil(t, toErrorDesc(nil, 400))
}

func TestStartStopManualPeriodicChecks(t *testing.T) {
	ckr := newChecker(healthCheckConfig{
		manualPeriodicCheckStart: true,
		checks: []*Check{
			{
				Name: "check",
				Check: func(ctx context.Context) error {
					return nil
				},
				refreshInterval: 50 * time.Minute,
			},
		}})

	assert.Equal(t, 0, len(ckr.endChans), "no goroutines must be started automatically")

	ckr.StartPeriodicChecks()
	assert.Equal(t, 1, len(ckr.endChans), "no goroutines were started on manual start")

	ckr.StopPeriodicChecks()
	assert.Equal(t, 0, len(ckr.endChans), "no goroutines were stopped on manual stop")
}

func TestStartAutomaticPeriodicChecks(t *testing.T) {
	ckr := newChecker(healthCheckConfig{
		manualPeriodicCheckStart: false,
		checks: []*Check{
			{
				Name: "check",
				Check: func(ctx context.Context) error {
					return nil
				},
				refreshInterval: 50 * time.Minute,
			},
		}})
	assert.Equal(t, 1, len(ckr.endChans), "no goroutines were started on manual start")

	ckr.StopPeriodicChecks()
	assert.Equal(t, 0, len(ckr.endChans), "no goroutines were stopped on manual stop")
}

func TestExecuteCheckFunc(t *testing.T) {
	// Arrange
	check := Check{
		Check: func(ctx context.Context) error {
			return nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Hour)
	defer cancel()

	// Act
	result := executeCheckFunc(ctx, &check)

	// Assert
	assert.Nil(t, result)
}

func TestExecuteCheckFuncWithTimeout(t *testing.T) {
	// Arrange
	check := Check{
		Check: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
			}
			return nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Act
	result := executeCheckFunc(ctx, &check)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, "check timed out", result.Error())
}

func TestInternalCheckWithCheckError(t *testing.T) {
	// Arrange
	check := Check{
		Check: func(ctx context.Context) error {
			return fmt.Errorf("ohi")
		},
	}
	state := checkState{
		startedAt:     time.Now().Add(-5 * time.Minute),
		lastCheckedAt: time.Now().Add(-5 * time.Minute),
		lastSuccessAt: time.Now().Add(-5 * time.Minute),
	}

	// Act
	result := doCheck(context.Background(), check, state)

	// Assert
	assert.Equal(t, true, state.lastCheckedAt.Before(result.newState.lastCheckedAt))
	assert.Equal(t, true, state.lastSuccessAt.Equal(result.newState.lastSuccessAt))
	assert.Equal(t, true, state.startedAt.Equal(result.newState.startedAt))
	assert.Equal(t, "UTC", result.newState.lastCheckedAt.Format("MST"))
	assert.Equal(t, uint(1), result.newState.consecutiveFails)
}

func TestInternalCheckWithCheckSuccess(t *testing.T) {
	// Arrange
	check := Check{
		Check: func(ctx context.Context) error {
			return nil
		},
	}
	state := checkState{
		startedAt:        time.Now().Add(-5 * time.Minute),
		lastCheckedAt:    time.Now().Add(-5 * time.Minute),
		lastSuccessAt:    time.Now().Add(-5 * time.Minute),
		consecutiveFails: 1000,
	}

	// Act
	result := doCheck(context.Background(), check, state)

	// Assert
	assert.Equal(t, true, state.lastCheckedAt.Before(result.newState.lastCheckedAt))
	assert.Equal(t, true, result.newState.lastCheckedAt.Equal(result.newState.lastCheckedAt))
	assert.Equal(t, true, state.startedAt.Equal(result.newState.startedAt))
	assert.Equal(t, "UTC", result.newState.lastCheckedAt.Format("MST"))
	assert.Equal(t, uint(0), result.newState.consecutiveFails)
}

func doTestCheckerCheckFunc(t *testing.T, refreshInterval time.Duration, err error, expectedStatus Status) {
	// Arrange
	checks := []*Check{
		{
			Name: "check1",
			Check: func(ctx context.Context) error {
				return nil
			},
		},
		{
			Name: "check2",
			Check: func(ctx context.Context) error {
				return err
			},
			refreshInterval: refreshInterval,
		},
	}

	ckr := newChecker(healthCheckConfig{checks: checks})

	// Act
	res := ckr.Check(context.Background())

	// Assert
	require.NotNil(t, res.Details)
	assert.Equal(t, res.Status, expectedStatus)
	for _, ck := range checks {
		_, checkResultExists := (*res.Details)[ck.Name]
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
