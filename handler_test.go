package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type checkerMock struct {
	mock.Mock
}

func (ck *checkerMock) Start() {
	ck.Called()
}

func (ck *checkerMock) Stop() {
	ck.Called()
}

func (ck *checkerMock) Check(ctx context.Context) AggregatedCheckStatus {
	args := ck.Called(ctx)
	return args.Get(0).(AggregatedCheckStatus)
}

func TestStartHandler(t *testing.T) {
	// Arrange
	ckr := checkerMock{}
	ckr.On("Start")
	ckr.On("Check", mock.Anything).Return(AggregatedCheckStatus{})
	handler := newHandler(healthCheckConfig{}, &ckr)

	// Act
	handler.Start()

	// Assert
	ckr.Mock.AssertCalled(t, "Start")
}

func TestStopHandler(t *testing.T) {
	// Arrange
	ckr := checkerMock{}
	ckr.On("Stop")
	handler := newHandler(healthCheckConfig{}, &ckr)

	// Act
	handler.Stop()

	// Assert
	ckr.Mock.AssertCalled(t, "Stop")
}

func doTestHandler(t *testing.T, cfg healthCheckConfig, expectedStatus AggregatedCheckStatus, expectedStatusCode int) {
	// Arrange
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "https://localhost/foo", nil)

	ckr := checkerMock{}
	ckr.On("Check", mock.Anything).Return(expectedStatus)

	handler := newHandler(cfg, &ckr)

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	ckr.Mock.AssertNumberOfCalls(t, "Check", 1)
	assert.Equal(t, response.Header().Get("content-type"), "application/json; charset=utf-8")
	assert.Equal(t, response.Result().StatusCode, expectedStatusCode)

	result := AggregatedCheckStatus{}
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	assert.True(t, reflect.DeepEqual(result, expectedStatus))
}

func TestHandlerIfCheckFailThenRespondWithNotAvailable(t *testing.T) {
	err := "hello"
	status := AggregatedCheckStatus{
		Status: StatusUnknown,
		Details: &map[string]CheckResult{
			"check1": {Status: StatusDown, Timestamp: time.Now().UTC(), Error: &err},
			"check2": {Status: StatusUp, Timestamp: time.Now().UTC(), Error: nil},
		},
	}

	// Use non-standard Status codes
	cfg := healthCheckConfig{statusCodeUp: http.StatusNoContent, statusCodeDown: http.StatusTeapot}

	doTestHandler(t, cfg, status, http.StatusTeapot)
}

func TestHandlerIfCheckSucceedsThenRespondWithAvailable(t *testing.T) {
	status := AggregatedCheckStatus{
		Status: StatusUp,
		Details: &map[string]CheckResult{
			"check1": {Status: StatusUp, Timestamp: time.Now().UTC(), Error: nil},
		},
	}

	// Use non-standard Status codes
	cfg := healthCheckConfig{statusCodeUp: http.StatusNoContent, statusCodeDown: http.StatusTeapot}

	doTestHandler(t, cfg, status, http.StatusNoContent)
}

func TestHandlerIfAuthFailsThenReturnNoDetails(t *testing.T) {
	err := "an error message"
	status := AggregatedCheckStatus{
		Status: StatusDown,
		Details: &map[string]CheckResult{
			"check1": {Status: StatusDown, Timestamp: time.Now().UTC(), Error: &err},
		},
	}

	// Use non-standard Status codes
	cfg := healthCheckConfig{statusCodeUp: http.StatusNoContent, statusCodeDown: http.StatusTeapot}

	doTestHandler(t, cfg, status, http.StatusTeapot)
}

func TestWithGlobalTimeout(t *testing.T) {
	// Arrange
	testStart := time.Now()
	deadline, ok := testStart, false
	cfg := healthCheckConfig{timeout: 5 * time.Hour}

	ckr := checkerMock{}
	ckr.On("Check", mock.Anything).
		Return(AggregatedCheckStatus{}).
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
