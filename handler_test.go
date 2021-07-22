package health

import (
	"context"
	"encoding/json"
	"log"
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

func (ck *checkerMock) Check(ctx context.Context) CheckerResult {
	return ck.Called(ctx).Get(0).(CheckerResult)
}

func (ck *checkerMock) GetRunningPeriodicCheckCount() int {
	return ck.Called().Get(0).(int)
}

func doTestHandler(t *testing.T, statusCodeUp, statusCodeDown int, expectedStatus CheckerResult, expectedStatusCode int) {
	// Arrange
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "https://localhost/foo", nil)

	ckr := checkerMock{}
	ckr.On("Start")
	ckr.On("Check", mock.Anything).Return(expectedStatus)

	handler := NewHandlerWithConfig(&ckr, HandlerConfig{statusCodeUp, statusCodeDown, false})

	// Act
	handler.ServeHTTP(response, request)

	// Assert
	ckr.Mock.AssertNumberOfCalls(t, "Check", 1)
	assert.Equal(t, response.Header().Get("content-type"), "application/json; charset=utf-8")
	assert.Equal(t, response.Result().StatusCode, expectedStatusCode)

	result := CheckerResult{}
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	log.Printf("returned %+v, want %+v", result.Details, expectedStatus.Details)

	assert.True(t, reflect.DeepEqual(result, expectedStatus))
}

func TestHandlerIfCheckFailThenRespondWithNotAvailable(t *testing.T) {
	now := time.Now().UTC()
	err := "hello"
	status := CheckerResult{
		Status: StatusUnknown,
		Details: &map[string]CheckResult{
			"check1": {Status: StatusDown, Timestamp: &now, Error: &err},
			"check2": {Status: StatusUp, Timestamp: &now, Error: nil},
		},
	}

	doTestHandler(t, http.StatusNoContent, http.StatusTeapot, status, http.StatusTeapot)
}

func TestHandlerIfCheckSucceedsThenRespondWithAvailable(t *testing.T) {
	now := time.Now().UTC()
	status := CheckerResult{
		Status: StatusUp,
		Details: &map[string]CheckResult{
			"check1": {Status: StatusUp, Timestamp: &now, Error: nil},
		},
	}

	doTestHandler(t, http.StatusNoContent, http.StatusTeapot, status, http.StatusNoContent)
}

func TestHandlerIfAuthFailsThenReturnNoDetails(t *testing.T) {
	now := time.Now().UTC()
	err := "an error message"
	status := CheckerResult{
		Status: StatusDown,
		Details: &map[string]CheckResult{
			"check1": {Status: StatusDown, Timestamp: &now, Error: &err},
		},
	}
	doTestHandler(t, http.StatusNoContent, http.StatusTeapot, status, http.StatusTeapot)
}
