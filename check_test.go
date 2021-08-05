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
		LastCheckedAt: &time.Time{},
	})
}

func TestWhenNoErrorThenStatusUp(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusUp, 0, 0, CheckState{
		LastCheckedAt: &now,
	})
}

func TestWhenErrorThenStatusDown(t *testing.T) {
	now := time.Now()
	doTestEvaluateAvailabilityStatus(t, StatusDown, 0, 0, CheckState{
		LastCheckedAt: &now,
		Result:        fmt.Errorf("example error"),
	})
}

func TestWhenErrorAndMaxFailuresThresholdNotCrossedThenStatusWarn(t *testing.T) {
	now := time.Now()
	lastSuccessAt := now.Add(-3 * time.Minute)

	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Second, uint(10), CheckState{
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
	doTestEvaluateAvailabilityStatus(t, StatusUp, 1*time.Hour, uint(1), CheckState{
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
	doTestEvaluateAvailabilityStatus(t, StatusDown, 1*time.Second, uint(1), CheckState{
		LastCheckedAt:       &now,
		Result:              fmt.Errorf("example error"),
		FirstCheckStartedAt: time.Now().Add(-3 * time.Minute),
		LastSuccessAt:       &lastSuccessAt,
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

func doTestCheckerCheckWithIgnoredFunc(t *testing.T, updateInterval time.Duration, err error, ignoreTwo bool, expectedStatus AvailabilityStatus) {
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
			Ignore: ignoreTwo,
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

func TestWhenOneIgnoredCheckFailedThenAggregatedResultUp(t *testing.T) {
	doTestCheckerCheckWithIgnoredFunc(t, 0, fmt.Errorf("this is a check error"), true, StatusUp)
}

func TestCheckSuccessNotAllChecksExecutedYet(t *testing.T) {
	doTestCheckerCheckFunc(t, 5*time.Hour, nil, StatusUnknown)
}

/*
The following tests should be removed and a more general test should be written that tests the
public interface of the checker rather than private functions.
*/

// TODO: Add test for createNextCheckState
// TODO: Add test for createNextCheckState
// TODO: Add test for mapStateToCheckerResult

/*
func TestToErrorDescErrorShortened(t *testing.T) {
	assert.Equal(t, "this", *toErrorDesc(fmt.Errorf("this is nice"), 4))
}

func TestToErrorDescErrorNotShortened(t *testing.T) {
	assert.Equal(t, "this is nice", *toErrorDesc(fmt.Errorf("this is nice"), 400))
}

func TestToErrorDescNoError(t *testing.T) {
	assert.Nil(t, toErrorDesc(nil, 400))
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

	var listener StatusListener = func(status AvailabilityStatus, state map[string]CheckState) {
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
	assert.Equal(t, expectedErr, (*actualResults)[expectedCheckName].Result)
	assert.Equal(t, StatusDown, (*actualResults)[expectedCheckName].CheckerResult)
}


func TestInternalCheckWithCheckSuccess(t *testing.T) {
	// Arrange
	check := Check{
		Check: func(ctx context.Context) error {
			return nil
		},
	}
	state := CheckState{
		FirstCheckedAt:   time.Now().Add(-5 * time.Minute),
		LastCheckedAt:    time.Now().Add(-5 * time.Minute),
		LastSuccessAt:    time.Now().Add(-5 * time.Minute),
		ContiguousFails: 1000,
	}

	// Act
	result := executeCheck(context.Background(), check, state)

	// Assert
	assert.Equal(t, true, state.LastCheckedAt.Before(result.newState.LastCheckedAt))
	assert.Equal(t, true, result.newState.LastCheckedAt.Equal(result.newState.LastCheckedAt))
	assert.Equal(t, true, state.FirstCheckedAt.Equal(result.newState.startedAt))
	assert.Equal(t, "UTC", result.newState.LastCheckedAt.Format("MST"))
	assert.Equal(t, uint(0), result.newState.ContiguousFails)
}


func TestInternalCheckWithCheckError(t *testing.T) {
	// Arrange
	check := Check{
		Check: func(ctx context.Context) error {
			return fmt.Errorf("ohi")
		},
	}
	state := CheckState{
		FirstCheckedAt: time.Now().Add(-5 * time.Minute),
		LastCheckedAt:  time.Now().Add(-5 * time.Minute),
		LastSuccessAt:  time.Now().Add(-5 * time.Minute),
	}

	// Act
	result := executeCheck(context.Background(), check, state)

	// Assert
	assert.Equal(t, true, state.LastCheckedAt.Before(result.newState.LastCheckedAt))
	assert.Equal(t, true, state.LastSuccessAt.Equal(result.newState.LastSuccessAt))
	assert.Equal(t, true, state.FirstCheckedAt.Equal(result.newState.startedAt))
	assert.Equal(t, "UTC", result.newState.LastCheckedAt.Format("MST"))
	assert.Equal(t, uint(1), result.newState.ContiguousFails)
}


func TestNewAggregatedCheckStatusWithDetails(t *testing.T) {
	// Arrange
	errMsg := "this is an error message"
	testData := map[string]CheckResult{"check1": {StatusDown, time.Now(), &errMsg}}

	// Act
	result := newCheckerResults(StatusDown, testData, true)

	// Assert
	assert.Equal(t, StatusDown, result.CheckerResult)
	assert.Equal(t, true, reflect.DeepEqual(&testData, result.Details))
}

func TestNewAggregatedCheckStatusWithoutDetails(t *testing.T) {
	// Arrange
	testData := map[string]CheckResult{}

	// Act
	result := newCheckerResults(StatusDown, testData, false)

	// Assert
	assert.Equal(t, StatusDown, result.CheckerResult)
	assert.Nil(t, result.Details)
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

*/
