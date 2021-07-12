package health

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type checkerMock struct {
	mock.Mock
}

func (s *availabilityStatus) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	*s = map[string]availabilityStatus{
		"UP":      statusUp,
		"WARN":    statusWarn,
		"UNKNOWN": statusUnknown,
		"DOWN":    statusDown,
	}[str]

	return nil
}

func (ck *checkerMock) StartPeriodicChecks() {
	ck.Called()
}

func (ck *checkerMock) StopPeriodicChecks() {
	ck.Called()
}

func (ck *checkerMock) Check(ctx context.Context) aggregatedCheckStatus {
	args := ck.Called(ctx)
	return args.Get(0).(aggregatedCheckStatus)
}

func TestStartPeriodicChecks(t *testing.T) {
	// Arrange
	ckr := checkerMock{}
	ckr.On("StartPeriodicChecks")
	handler := newHandler(healthCheckConfig{}, &ckr)

	// Act
	StartPeriodicChecks(handler)

	// Assert
	ckr.Mock.AssertCalled(t, "StartPeriodicChecks")
}

func TestStopPeriodicChecks(t *testing.T) {
	// Arrange
	ckr := checkerMock{}
	ckr.On("StopPeriodicChecks")
	handler := newHandler(healthCheckConfig{}, &ckr)

	// Act
	StopPeriodicChecks(handler)

	// Assert
	ckr.Mock.AssertCalled(t, "StopPeriodicChecks")
}

func doTestHandler(t *testing.T, expectedStatus aggregatedCheckStatus, expectedStatusCode int) {
	// Arrange
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "https://localhost/foo", nil)

	ckr := checkerMock{}
	ckr.On("Check", mock.Anything).Return(expectedStatus)

	handler := newHandler(healthCheckConfig{}, &ckr)

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	ckr.Mock.AssertNumberOfCalls(t, "Check", 1)
	assert.Equal(t, response.Header().Get("content-type"), "application/json; charset=utf-8")
	assert.Equal(t, response.Result().StatusCode, expectedStatusCode)

	result := aggregatedCheckStatus{}
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	assert.True(t, reflect.DeepEqual(result, expectedStatus))
}

func TestHandlerIfCheckFailThenRespondWithNotAvailable(t *testing.T) {
	ts := time.Now().UTC()
	err := "hello"

	status := aggregatedCheckStatus{
		Status:    statusUnknown,
		Timestamp: &ts,
		Details: &map[string]checkStatus{
			"check1": {Status: statusDown, Timestamp: time.Now().UTC(), Error: &err},
			"check2": {Status: statusUp, Timestamp: time.Now().UTC(), Error: nil},
		},
	}

	doTestHandler(t, status, 503)
}

func TestHandlerIfCheckSucceedsThenRespondWithAvailable(t *testing.T) {
	ts := time.Now().UTC()
	status := aggregatedCheckStatus{
		Status:    statusUp,
		Timestamp: &ts,
		Details: &map[string]checkStatus{
			"check1": {Status: statusUp, Timestamp: time.Now().UTC(), Error: nil},
		},
	}

	doTestHandler(t, status, 200)
}

func TestHandlerIfAuthFailsThenReturnNoDetails(t *testing.T) {
	ts := time.Now().UTC()
	err := "an error message"

	status := aggregatedCheckStatus{
		Status:    statusDown,
		Timestamp: &ts,
		Details: &map[string]checkStatus{
			"check1": {Status: statusDown, Timestamp: time.Now().UTC(), Error: &err},
		},
	}

	doTestHandler(t, status, 503)
}

func TestWithGlobalTimeout(t *testing.T) {
	// Arrange
	testStart := time.Now()
	deadline, ok := testStart, false
	cfg := healthCheckConfig{timeout: 5 * time.Hour}

	ckr := checkerMock{}
	ckr.On("Check", mock.Anything).
		Return(aggregatedCheckStatus{}).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			deadline, ok = ctx.Deadline()
		})
	handler := newHandler(cfg, &ckr)

	req := httptest.NewRequest("GET", "https://localhost/foo", nil)
	res := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(res, req)

	// Assert
	ckr.Mock.AssertCalled(t, "Check", mock.Anything)

	assert.True(t, ok)
	assert.True(t, deadline.After(testStart.Add(cfg.timeout)))
}
