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
	check := Check{Name: "test"}

	// Act
	WithCheck(check)(&cfg)

	// Assert
	require.Equal(t, 1, len(cfg.checks))
	assert.True(t, reflect.DeepEqual(&check, cfg.checks[0]))
}

func TestWithCacheDurationConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}
	duration := 5 * time.Hour

	// Act
	WithCache(duration)(&cfg)

	// Assert
	assert.Equal(t, duration, cfg.cacheTTL)
}

func TestWithCacheConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithCache(1 * time.Second)(&cfg)

	// Assert
	assert.Equal(t, 1*time.Second, cfg.cacheTTL)
}

func TestWithManualPeriodicCheckStartConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithManualPeriodicCheckStart()(&cfg)

	// Assert
	assert.True(t, cfg.manualPeriodicCheckStart)
}

func TestAuthMiddlewareConfig(t *testing.T) {
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

func TestWithMaxErrorMessageLengthConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithMaxErrorMessageLength(300)(&cfg)

	// Assert
	assert.Equal(t, uint(300), cfg.maxErrMsgLen)
}

func TestNewWithDefaults(t *testing.T) {
	// Arrange
	configApplied := false
	opt := func(config *healthCheckConfig) { configApplied = true }

	// Act
	handler := NewHandler(opt)

	// Assert
	ckr := handler.(*healthCheckHandler).ckr.(*defaultChecker)
	assert.Equal(t, time.Duration(0), ckr.cfg.cacheTTL)
	assert.Equal(t, 30*time.Second, ckr.cfg.timeout)
	assert.Equal(t, uint(500), ckr.cfg.maxErrMsgLen)
	assert.True(t, configApplied)
}
