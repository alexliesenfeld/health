package health

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestWithPeriodicCheckConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}
	check := Check{Name: "test"}
	interval := 5 * time.Second

	// Act
	WithPeriodicCheck(interval, check)(&cfg)
	check.refreshInterval = interval

	// Assert
	assert.Equal(t, 1, len(cfg.checks))
	assert.True(t, reflect.DeepEqual(check, *cfg.checks[0]))
}

func TestWithCheckConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}
	duration := 5 * time.Hour

	// Act
	WithCacheDuration(duration)(&cfg)

	// Assert
	assert.Equal(t, duration, cfg.cacheDuration)
}

func TestWithDisabledCacheConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithDisabledCache()(&cfg)

	// Assert
	assert.Equal(t, 0*time.Second, cfg.cacheDuration)
}

func TestWithManualPeriodicCheckStartConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithManualPeriodicCheckStart()(&cfg)

	// Assert
	assert.True(t, cfg.manualPeriodicCheckStart)
}

func TestWithTimeoutConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}
	duration := 5 * time.Hour
	testStart := time.Now()

	r := httptest.NewRequest("GET", "https://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Act
	WithTimeout(duration)(&cfg)

	// Assert
	require.Equal(t, 1, len(cfg.middleware))

	// Arrange 2
	deadline, ok := time.Now(), false

	// Act 2
	cfg.middleware[0](http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		deadline, ok = request.Context().Deadline()
	})).ServeHTTP(w, r)

	// Assert 2
	assert.True(t, ok)
	assert.True(t, deadline.After(testStart.Add(duration)))
}
