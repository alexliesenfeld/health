package health

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
)

func TestStatusUnknownBeforeStatusUp(t *testing.T) {
	// Arrange
	testData := map[string]CheckState{
		"check1": {Status: StatusUp},
		"check2": {Status: StatusUnknown},
	}

	// Act
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, result, StatusUnknown)
}

func TestStatusDownBeforeStatusUnknown(t *testing.T) {
	// Arrange
	testData := map[string]CheckState{
		"check1": {Status: StatusDown},
		"check2": {Status: StatusUnknown},
	}

	// Act
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, result, StatusDown)
}

func TestNewAggregatedCheckStatusWithDetails(t *testing.T) {
	// Arrange
	errMsg := "this is an error message"
	testData := map[string]CheckStatus{"check1": {StatusDown, time.Now(), &errMsg}}

	// Act
	result := newSystemStatus(StatusDown, testData, true)

	// Assert
	assert.Equal(t, StatusDown, result.Status)
	assert.Equal(t, true, reflect.DeepEqual(&testData, result.Details))
}

func TestNewAggregatedCheckStatusWithoutDetails(t *testing.T) {
	// Arrange
	testData := map[string]CheckStatus{}

	// Act
	result := newSystemStatus(StatusDown, testData, false)

	// Assert
	assert.Equal(t, StatusDown, result.Status)
	assert.Nil(t, result.Details)
}

func doTestEvaluateAvailabilityStatus(
	t *testing.T,
	expectedStatus AvailabilityStatus,
	maxTimeInError time.Duration,
	maxFails uint,
	state CheckState,
) {
	// Act
	result := evaluateCheckStatus(&state, maxTimeInError, maxFails)

	// Assert
	assert.Equal(t, expectedStatus, result)
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
		LastResult:    fmt.Errorf("example error"),
	})
}

func TestWhenErrorAndMaxFailuresThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Second, uint(10), CheckState{
		LastCheckedAt:    time.Now(),
		LastResult:       fmt.Errorf("example error"),
		startedAt:        time.Now().Add(-3 * time.Minute),
		LastSuccessAt:    time.Now().Add(-2 * time.Minute),
		ConsecutiveFails: 1,
	})
}

func TestWhenErrorAndMaxTimeInErrorThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Hour, uint(1), CheckState{
		LastCheckedAt:    time.Now(),
		LastResult:       fmt.Errorf("example error"), // Required for the test
		startedAt:        time.Now().Add(-3 * time.Minute),
		LastSuccessAt:    time.Now().Add(-2 * time.Minute),
		ConsecutiveFails: 100,
	})
}

func TestWhenErrorAndAllThresholdsCrossedThenStatusDown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, StatusDown, 1*time.Second, uint(1), CheckState{
		LastCheckedAt:    time.Now(),
		LastResult:       fmt.Errorf("example error"), // Required for the test
		startedAt:        time.Now().Add(-3 * time.Minute),
		LastSuccessAt:    time.Now().Add(-2 * time.Minute),
		ConsecutiveFails: 5,
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
	ckr := newDefaultChecker(healthCheckConfig{
		withManualStart: true,
		checks: map[string]*Check{
			"check": {
				Name: "check",
				Check: func(ctx context.Context) error {
					return nil
				},
				updateInterval: 50 * time.Minute,
			},
		}})

	assert.Equal(t, 0, len(ckr.endChans), "no goroutines must be started automatically")

	ckr.Start()
	assert.Equal(t, 1, len(ckr.endChans), "no goroutines were started on manual start")

	ckr.Stop()
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
	state := CheckState{
		startedAt:     time.Now().Add(-5 * time.Minute),
		LastCheckedAt: time.Now().Add(-5 * time.Minute),
		LastSuccessAt: time.Now().Add(-5 * time.Minute),
	}

	// Act
	result := doCheck(context.Background(), check, state)

	// Assert
	assert.Equal(t, true, state.LastCheckedAt.Before(result.newState.LastCheckedAt))
	assert.Equal(t, true, state.LastSuccessAt.Equal(result.newState.LastSuccessAt))
	assert.Equal(t, true, state.startedAt.Equal(result.newState.startedAt))
	assert.Equal(t, "UTC", result.newState.LastCheckedAt.Format("MST"))
	assert.Equal(t, uint(1), result.newState.ConsecutiveFails)
}

func TestInternalCheckWithCheckSuccess(t *testing.T) {
	// Arrange
	check := Check{
		Check: func(ctx context.Context) error {
			return nil
		},
	}
	state := CheckState{
		startedAt:        time.Now().Add(-5 * time.Minute),
		LastCheckedAt:    time.Now().Add(-5 * time.Minute),
		LastSuccessAt:    time.Now().Add(-5 * time.Minute),
		ConsecutiveFails: 1000,
	}

	// Act
	result := doCheck(context.Background(), check, state)

	// Assert
	assert.Equal(t, true, state.LastCheckedAt.Before(result.newState.LastCheckedAt))
	assert.Equal(t, true, result.newState.LastCheckedAt.Equal(result.newState.LastCheckedAt))
	assert.Equal(t, true, state.startedAt.Equal(result.newState.startedAt))
	assert.Equal(t, "UTC", result.newState.LastCheckedAt.Format("MST"))
	assert.Equal(t, uint(0), result.newState.ConsecutiveFails)
}

func doTestCheckerCheckFunc(t *testing.T, updateInterval time.Duration, err error, expectedStatus AvailabilityStatus) {
	// Arrange
	checks := map[string]*Check{
		"check1": {
			Name: "check1",
			Check: func(ctx context.Context) error {
				return nil
			},
		},
		"check2": {
			Name: "check2",
			Check: func(ctx context.Context) error {
				return err
			},
			updateInterval: updateInterval,
		},
	}

	ckr := newDefaultChecker(healthCheckConfig{checks: checks, timeout: 10 * time.Second})

	// Act
	res := ckr.Check(context.Background())

	// Assert
	require.NotNil(t, res.Details)
	assert.Equal(t, expectedStatus, res.Status)
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

func TestCheckExecuteListeners(t *testing.T) {
	// Arrange
	var (
		actualStatus      *AvailabilityStatus    = nil
		actualResults     *map[string]CheckState = nil
		expectedErr                              = fmt.Errorf("test error")
		expectedCheckName                        = "testCheck"
	)

	checks := map[string]*Check{
		expectedCheckName: {
			Name: expectedCheckName,
			Check: func(ctx context.Context) error {
				return expectedErr
			},
		},
	}

	var listener SystemStatusListener = func(status AvailabilityStatus, state map[string]CheckState) {
		actualStatus = &status
		actualResults = &state
	}

	ckr := newDefaultChecker(healthCheckConfig{
		checks:               checks,
		statusChangeListener: listener,
		maxErrMsgLen:         10,
		timeout:              10 * time.Second,
	})

	// Act
	ckr.Check(context.Background())

	// Assert
	assert.Equal(t, StatusDown, *actualStatus)
	assert.Equal(t, 1, len(*actualResults))
	assert.Equal(t, expectedErr, (*actualResults)[expectedCheckName].LastResult)
	assert.Equal(t, StatusDown, (*actualResults)[expectedCheckName].Status)
}

// TODO: Add test for updateState
// TODO: Add test for updateCheckState
// TODO: Add test for mapStateToCheckStatus
