package health

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestAggregateResult(t *testing.T) {
	// Arrange
	errMsg := "this is an error message"
	testData := map[string]checkStatus{
		"check1": {
			Status:    statusUp,
			Error:     nil,
			Timestamp: time.Now().Add(-5 * time.Minute),
		},
		"check2": {
			Status:    statusWarn,
			Error:     nil,
			Timestamp: time.Now().Add(-3 * time.Minute),
		},
		"check3": {
			Status:    statusDown,
			Error:     &errMsg,
			Timestamp: time.Now().Add(-1 * time.Minute),
		},
	}

	// Act
	result := aggregateStatus(testData, true)

	// Assert
	assert.Equal(t, statusDown, result.Status)
	assert.Equal(t, true, result.Timestamp.Equal(testData["check1"].Timestamp))
	assert.Equal(t, true, reflect.DeepEqual(&testData, result.Details))
}

func TestAggregateResultWithoutDetails(t *testing.T) {
	// Arrange
	testData := map[string]checkStatus{"check1": {Status: statusUp, Timestamp: time.Now()}}

	// Act
	result := aggregateStatus(testData, false)

	// Assert
	assert.Equal(t, statusUp, result.Status)
	assert.Nil(t, result.Timestamp)
	assert.Nil(t, result.Details)
}

func doTestEvaluateAvailabilityStatus(
	t *testing.T,
	expectedStatus availabilityStatus,
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
	doTestEvaluateAvailabilityStatus(t, statusUnknown, 0, 0, checkState{
		lastCheckedAt: time.Time{},
	})
}

func TestWhenNoErrorThenStatusUp(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, statusUp, 0, 0, checkState{
		lastCheckedAt: time.Now(),
	})
}

func TestWhenErrorThenStatusDown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, statusDown, 0, 0, checkState{
		lastCheckedAt: time.Now(),
		lastResult:    fmt.Errorf("example error"),
	})
}

func TestWhenErrorAndMaxFailuresThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, statusWarn, 1*time.Second, uint(10), checkState{
		lastCheckedAt:    time.Now(),
		lastResult:       fmt.Errorf("example error"),
		startedAt:        time.Now().Add(-3 * time.Minute),
		lastSuccessAt:    time.Now().Add(-2 * time.Minute),
		consecutiveFails: 1,
	})
}

func TestWhenErrorAndMaxTimeInErrorThresholdNotCrossedThenStatusWarn(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, statusWarn, 1*time.Hour, uint(1), checkState{
		lastCheckedAt:    time.Now(),
		lastResult:       fmt.Errorf("example error"), // Required for the test
		startedAt:        time.Now().Add(-3 * time.Minute),
		lastSuccessAt:    time.Now().Add(-2 * time.Minute),
		consecutiveFails: 100,
	})
}

func TestWhenErrorAndAllThresholdsCrossedThenStatusDown(t *testing.T) {
	doTestEvaluateAvailabilityStatus(t, statusDown, 1*time.Second, uint(1), checkState{
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
	handler := New(
		WithManualPeriodicCheckStart(),
		WithPeriodicCheck(50*time.Minute, Check{
			Name: "check",
			Check: func(ctx context.Context) error {
				return nil
			},
		})).(*healthCheckHandler)
	assert.Equal(t, 0, len(handler.ckr.endChans), "no goroutines must be started automatically")

	StartPeriodicChecks(handler)
	assert.Equal(t, 1, len(handler.ckr.endChans), "no goroutines were started on manual start")

	StopPeriodicChecks(handler)
	assert.Equal(t, 0, len(handler.ckr.endChans), "no goroutines were stopped on manual stop")
}

func TestStartAutomaticPeriodicChecks(t *testing.T) {
	handler := New(
		WithPeriodicCheck(50*time.Minute, Check{
			Name: "check",
			Check: func(ctx context.Context) error {
				return nil
			},
		})).(*healthCheckHandler)
	assert.Equal(t, 1, len(handler.ckr.endChans), "no goroutines were started on manual start")

	StopPeriodicChecks(handler)
	assert.Equal(t, 0, len(handler.ckr.endChans), "no goroutines were stopped on manual stop")
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
