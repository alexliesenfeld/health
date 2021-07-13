package health

import (
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
	WithCacheDuration(duration)(&cfg)

	// Assert
	assert.Equal(t, duration, cfg.cacheTTL)
}

func TestWithDisabledCacheConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithDisabledCache()(&cfg)

	// Assert
	assert.Equal(t, 0*time.Second, cfg.cacheTTL)
}

func TestWithManualPeriodicCheckStartConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithManualPeriodicCheckStart()(&cfg)

	// Assert
	assert.True(t, cfg.manualPeriodicCheckStart)
}

func TestWithTimeoutStartConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithTimeout(5 * time.Hour)(&cfg)

	// Assert
	assert.Equal(t, 5*time.Hour, cfg.timeout)
}

func TestWithMaxErrorMessageLengthConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	WithMaxErrorMessageLength(300)(&cfg)

	// Assert
	assert.Equal(t, uint(300), cfg.maxErrMsgLen)
}

func TestWithCustomStatusCodesConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	// Use of non standard status codes.
	WithCustomStatusCodes(http.StatusCreated, http.StatusBadGateway)(&cfg)

	// Assert
	assert.Equal(t, http.StatusCreated, cfg.statusCodeUp)
	assert.Equal(t, http.StatusBadGateway, cfg.statusCodeDown)
}

func TestWithStatusChangeListenerConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	// Use of non standard status codes.
	WithStatusChangeListener(func(status Status, checks map[string]CheckResult) {})(&cfg)
	WithStatusChangeListener(func(status Status, checks map[string]CheckResult) {})(&cfg)

	// Assert
	assert.Equal(t, 2, len(cfg.statusChangeListeners))
	// Not possible in Go to compare functions.
}

func TestNewWithDefaults(t *testing.T) {
	// Arrange
	configApplied := false
	opt := func(config *healthCheckConfig) { configApplied = true }

	// Act
	handler := NewHandler(opt)

	// Assert
	ckr := handler.(*healthCheckHandler).ckr.(*defaultChecker)
	assert.Equal(t, 1*time.Second, ckr.cfg.cacheTTL)
	assert.Equal(t, 30*time.Second, ckr.cfg.timeout)
	assert.Equal(t, uint(500), ckr.cfg.maxErrMsgLen)
	assert.True(t, configApplied)
}
