package health

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutMiddleware(t *testing.T) {
	// Arrange
	deadline, ok := time.Now(), false
	testStart := time.Now()
	timeoutDuration := 5 * time.Hour

	req := httptest.NewRequest("GET", "https://localhost/foo", nil)
	res := httptest.NewRecorder()

	// Act
	newTimeoutMiddleware(timeoutDuration)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		deadline, ok = request.Context().Deadline()
	})).ServeHTTP(res, req)

	// Assert
	assert.True(t, ok)
	assert.True(t, deadline.After(testStart.Add(timeoutDuration)))
}

func TestAuthMiddlewareAuthSuccess(t *testing.T) {
	expectedAuthRes := true
	doTestAuthMiddleware(t, nil, false, true, &expectedAuthRes)
}

func TestAuthMiddlewareAuthFailure(t *testing.T) {
	doTestAuthMiddleware(t, fmt.Errorf("my auth error"), false, false, nil)
}

func TestAuthMiddlewareAuthFailureWithAuthStatus(t *testing.T) {
	expectedAuthRes := false
	doTestAuthMiddleware(t, fmt.Errorf("my auth error"), true, true, &expectedAuthRes)
}

func doTestAuthMiddleware(t *testing.T, err error, sendStatusOnAuthFailure, expectExecuted bool, expectAuthResult *bool) {
	// Arrange
	var (
		req                     = httptest.NewRequest("GET", "https://localhost/foo", nil)
		res                     = httptest.NewRecorder()
		innerFuncExecuted       = false
		authResult        *bool = nil
	)

	// Act
	newAuthMiddleware(sendStatusOnAuthFailure, func(r *http.Request) error {
		return err
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerFuncExecuted = true
		authResult = getAuthResult(r.Context())
	})).ServeHTTP(res, req)

	// Assert
	assert.Equal(t, expectExecuted, innerFuncExecuted)
	assert.Equal(t, expectAuthResult, authResult)
}

func TestBasicAuthMiddlewareSuccess(t *testing.T) {
	expectedAuthRes := true
	doTestBasicAuthMiddleware(t, "user", "pw", "pw", false, true, &expectedAuthRes)
}

func TestBasicAuthMiddlewareFailure(t *testing.T) {
	doTestBasicAuthMiddleware(t, "user", "pw", "wrong", false, false, nil)
}

func TestBasicAuthMiddlewareFailureWithAuthStatus(t *testing.T) {
	expectedAuthRes := false
	doTestBasicAuthMiddleware(t, "user", "pw", "wrong", true, true, &expectedAuthRes)
}

func doTestBasicAuthMiddleware(t *testing.T,
	user, password, expectPassword string,
	sendStatusOnAuthError, expectExecuted bool,
	expectAuthRes *bool,
) {
	// Arrange
	var (
		req                     = httptest.NewRequest("GET", "https://localhost/foo", nil)
		res                     = httptest.NewRecorder()
		innerFuncExecuted       = false
		authResult        *bool = nil
	)
	req.SetBasicAuth(user, password)

	// Act
	mw := newBasicAuthMiddleware(user, expectPassword, sendStatusOnAuthError)
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerFuncExecuted = true
		authResult = getAuthResult(r.Context())
	})).ServeHTTP(res, req)

	// Assert
	assert.Equal(t, innerFuncExecuted, expectExecuted)
	assert.Equal(t, expectAuthRes, authResult)
}
