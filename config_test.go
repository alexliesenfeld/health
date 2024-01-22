package health

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithPeriodicCheckConfig(t *testing.T) {
	// Arrange
	expectedName := "test"
	cfg := checkerConfig{syncChecks: map[string]*Check{}, asyncChecks: map[string]asyncCheck{}}
	interval := 5 * time.Second
	initialDelay := 1 * time.Minute
	check := Check{Name: expectedName}

	// Act
	WithPeriodicCheck(interval, initialDelay, check)(&cfg)

	// Assert
	assert.Equal(t, 1, len(cfg.asyncChecks))
	if p, ok := cfg.asyncChecks[expectedName].(*periodicCheck); assert.True(t, ok) {
		assert.True(t, reflect.DeepEqual(check, p.Check))
	}
}

func TestWithCheckConfig(t *testing.T) {
	// Arrange
	expectedName := "test"
	cfg := checkerConfig{syncChecks: map[string]*Check{}}
	check := Check{Name: "test"}

	// Act
	WithCheck(check)(&cfg)

	// Assert
	require.Equal(t, 1, len(cfg.syncChecks))
	assert.True(t, reflect.DeepEqual(&check, cfg.syncChecks[expectedName]))
}

func TestWithCacheDurationConfig(t *testing.T) {
	// Arrange
	cfg := checkerConfig{}
	duration := 5 * time.Hour

	// Act
	WithCacheDuration(duration)(&cfg)

	// Assert
	assert.Equal(t, duration, cfg.cacheTTL)
}

func TestWithDisabledCacheConfig(t *testing.T) {
	// Arrange
	cfg := checkerConfig{}

	// Act
	WithDisabledCache()(&cfg)

	// Assert
	assert.Equal(t, 0*time.Second, cfg.cacheTTL)
}

func TestWithTimeoutStartConfig(t *testing.T) {
	// Arrange
	cfg := checkerConfig{}

	// Act
	WithTimeout(5 * time.Hour)(&cfg)

	// Assert
	assert.Equal(t, 5*time.Hour, cfg.timeout)
}

func TestWithDisabledDetailsConfig(t *testing.T) {
	// Arrange
	cfg := checkerConfig{}

	// Act
	WithDisabledDetails()(&cfg)

	// Assert
	assert.Equal(t, true, cfg.detailsDisabled)
}

func TestWithMiddlewareConfig(t *testing.T) {
	// Arrange
	cfg := HandlerConfig{}
	mw := func(MiddlewareFunc) MiddlewareFunc {
		return func(r *http.Request) CheckerResult {
			return CheckerResult{nil, StatusUp, nil}
		}
	}

	// Act
	WithMiddleware(mw)(&cfg)

	// Assert
	assert.Equal(t, 1, len(cfg.middleware))
}

func TestWithInterceptorConfig(t *testing.T) {
	// Arrange
	cfg := checkerConfig{}
	interceptor := func(InterceptorFunc) InterceptorFunc {
		return func(ctx context.Context, name string, state CheckState) CheckState {
			return CheckState{}
		}
	}

	// Act
	WithInterceptors(interceptor)(&cfg)

	// Assert
	assert.Equal(t, 1, len(cfg.interceptors))
}

func TestWithResultWriterConfig(t *testing.T) {
	// Arrange
	cfg := HandlerConfig{}
	w := resultWriterMock{}

	// Act
	WithResultWriter(&w)(&cfg)

	// Assert
	assert.Equal(t, &w, cfg.resultWriter)
}

func TestWithStatusChangeListenerConfig(t *testing.T) {
	// Arrange
	cfg := checkerConfig{}

	// Act
	// Use of non standard AvailabilityStatus codes.
	WithStatusListener(func(ctx context.Context, state CheckerState) {})(&cfg)

	// Assert
	assert.NotNil(t, cfg.statusChangeListener)
	// Not possible in Go to compare functions.
}

func TestNewWithDefaults(t *testing.T) {
	// Arrange
	configApplied := false
	opt := func(config *checkerConfig) { configApplied = true }

	// Act
	checker := NewChecker(opt)

	// Assert
	ckr := checker.(*defaultChecker)
	assert.Equal(t, 1*time.Second, ckr.cfg.cacheTTL)
	assert.Equal(t, 10*time.Second, ckr.cfg.timeout)
	assert.True(t, configApplied)
}

func TestNewCheckerWithDefaults(t *testing.T) {
	// Arrange
	configApplied := false
	opt := func(config *checkerConfig) { configApplied = true }

	// Act
	checker := NewChecker(opt)

	// Assert
	ckr := checker.(*defaultChecker)
	assert.Equal(t, 1*time.Second, ckr.cfg.cacheTTL)
	assert.Equal(t, 10*time.Second, ckr.cfg.timeout)
	assert.True(t, configApplied)
}

func TestCheckerAutostartConfig(t *testing.T) {
	// Arrange + Act
	c := NewChecker()

	// Assert
	assert.True(t, c.IsStarted())
}

func TestCheckerAutostartDisabledConfig(t *testing.T) {
	// Arrange
	c := NewChecker(WithDisabledAutostart())

	// Assert
	assert.False(t, c.IsStarted())
}
