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

func (ck *checkerMock) Check(ctx context.Context, includeDetails bool) aggregatedCheckStatus {
	args := ck.Called(ctx, includeDetails)
	return args.Get(0).(aggregatedCheckStatus)
}

func TestStartPeriodicChecks(t *testing.T) {
	// Arrange
	ckr := checkerMock{}
	ckr.On("StartPeriodicChecks")
	handler := newHandler([]Middleware{}, &ckr)

	// Act
	StartPeriodicChecks(handler)

	// Assert
	ckr.Mock.AssertCalled(t, "StartPeriodicChecks")
}

func TestStopPeriodicChecks(t *testing.T) {
	// Arrange
	ckr := checkerMock{}
	ckr.On("StopPeriodicChecks")
	handler := newHandler([]Middleware{}, &ckr)

	// Act
	StopPeriodicChecks(handler)

	// Assert
	ckr.Mock.AssertCalled(t, "StopPeriodicChecks")
}

func doTestHandler(t *testing.T, authResult bool, expectedStatus aggregatedCheckStatus, expectedStatusCode int) {
	// Arrange
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "https://localhost/foo", nil)
	request = request.WithContext(withAuthResult(request.Context(), authResult))

	ckr := checkerMock{}
	ckr.On("Check", mock.Anything, authResult).Return(expectedStatus)

	handler := newHandler([]Middleware{}, &ckr)

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

	doTestHandler(t, true, status, 503)
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

	doTestHandler(t, true, status, 200)
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

	doTestHandler(t, false, status, 503)
}
