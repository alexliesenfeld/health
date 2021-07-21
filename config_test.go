package health

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithPeriodicCheckConfig(t *testing.T) {
	// Arrange
	expectedName := "test"
	cfg := healthCheckConfig{checks: map[string]*Check{}}
	check := Check{Name: expectedName}
	interval := 5 * time.Second

	// Act
	WithPeriodicCheck(interval, check)(&cfg)
	check.updateInterval = interval

	// Assert
	assert.Equal(t, 1, len(cfg.checks))
	assert.True(t, reflect.DeepEqual(check, *cfg.checks[expectedName]))
}

func TestWithCheckConfig(t *testing.T) {
	// Arrange
	expectedName := "test"
	cfg := healthCheckConfig{checks: map[string]*Check{}}
	check := Check{Name: "test"}

	// Act
	WithCheck(check)(&cfg)

	// Assert
	require.Equal(t, 1, len(cfg.checks))
	assert.True(t, reflect.DeepEqual(&check, cfg.checks[expectedName]))
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

func TestWithStatusChangeListenerConfig(t *testing.T) {
	// Arrange
	cfg := healthCheckConfig{}

	// Act
	// Use of non standard AvailabilityStatus codes.
	WithStatusListener(func(ctx context.Context, state CheckerState) context.Context { return nil })(&cfg)

	// Assert
	assert.NotNil(t, cfg.statusChangeListener)
	// Not possible in Go to compare functions.
}

func TestNewWithDefaults(t *testing.T) {
	// Arrange
	configApplied := false
	opt := func(config *healthCheckConfig) { configApplied = true }

	// Act
	checker := NewChecker(opt)

	// Assert
	ckr := checker.(*defaultChecker)
	assert.Equal(t, 1*time.Second, ckr.cfg.cacheTTL)
	assert.Equal(t, 30*time.Second, ckr.cfg.timeout)
	assert.Equal(t, uint(500), ckr.cfg.maxErrMsgLen)
	assert.True(t, configApplied)
}

func TestNewCheckerWithDefaults(t *testing.T) {
	// Arrange
	configApplied := false
	opt := func(config *healthCheckConfig) { configApplied = true }

	// Act
	checker := NewChecker(opt)

	// Assert
	ckr := checker.(*defaultChecker)
	assert.Equal(t, 1*time.Second, ckr.cfg.cacheTTL)
	assert.Equal(t, 30*time.Second, ckr.cfg.timeout)
	assert.Equal(t, uint(500), ckr.cfg.maxErrMsgLen)
	assert.True(t, configApplied)
}
