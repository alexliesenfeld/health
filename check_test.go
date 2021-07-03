package health

import (
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
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, statusDown, result.Status)
	assert.Equal(t, true, result.Timestamp.Equal(testData["check1"].Timestamp))
	assert.Equal(t, true, reflect.DeepEqual(testData, result.Checks))
	assert.Nil(t, result.Error)
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
