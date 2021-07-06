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

	r := httptest.NewRequest("GET", "https://localhost/foo", nil)
	w := httptest.NewRecorder()

	// Act
	newTimeoutMiddleware(timeoutDuration)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		deadline, ok = request.Context().Deadline()
	})).ServeHTTP(w, r)

	// Assert
	assert.True(t, ok)
	assert.True(t, deadline.After(testStart.Add(timeoutDuration)))
}

// TODO
func TestBasicAuthMiddleware(t *testing.T) {
	// Arrange
	r := httptest.NewRequest("GET", "https://localhost/foo", nil)
	w := httptest.NewRecorder()
	// TODO

	// Act
	newAuthMiddleware(true, func(r *http.Request) error {
		return fmt.Errorf("my test error")
	})(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// TODO
	})).ServeHTTP(w, r)

	// Assert
	// TODO
}
