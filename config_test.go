package health

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
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

func TestMiddlewareConfig(t *testing.T) {
	// Attention: This test function only tests the configuration aspect.
	// Testing the actual middleware can be found in separate tests.

	// Arrange
	options := []option{
		WithTimeout(5 * time.Hour),
		WithBasicAuth("peter", "pan", true),
		WithCustomAuth(true, func(r *http.Request) error {
			return fmt.Errorf("auth error")
		}),
	}

	for _, opt := range options {
		cfg := healthCheckConfig{}

		// Act
		opt(&cfg)

		// Assert
		require.Equal(t, 1, len(cfg.middleware))
		// TODO: Refactor, so you are able to assert that the correct middleware has been configured.
	}
}
