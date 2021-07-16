package health

import (
	"context"
	"net/http"
	"time"
)

type (
	// Check allows to configure health checks.
	Check struct {
		// The Name must be unique among all checks. Name is a required attribute.
		Name string
		// Check is the check function that will be executed to check availability.
		// This function must return an error if the checked service is considered
		// not available. Check is a required attribute.
		Check func(ctx context.Context) error
		// Timeout will override the global timeout value, if it is smaller than
		// the global timeout (see WithTimeout).
		Timeout time.Duration
		// FailureTolerance will set a duration for how long a service must be
		// "in error" until it is considered unavailable.
		FailureTolerance time.Duration
		// FailureToleranceThreshold will set a maximum number of consecutive
		// check fails until the service is considered unavailable.
		FailureToleranceThreshold uint
		// StatusListener allows to set a listener that will be called
		// whenever the Status of the check changes.
		StatusListener CheckStatusListener
		updateInterval time.Duration
	}

	option func(*healthCheckConfig)
)

// NewHandler creates a new health check http.Handler. If periodic checks have
// been configured (see WithPeriodicCheck), they will be started as well
// (if not explicitly turned off using WithManualStart).
func NewHandler(options ...option) Handler {
	cfg := healthCheckConfig{
		statusCodeUp:   http.StatusOK,
		statusCodeDown: http.StatusServiceUnavailable,
		cacheTTL:       1 * time.Second,
		timeout:        30 * time.Second,
		maxErrMsgLen:   500,
		checks:         map[string]*Check{},
	}

	for _, opt := range options {
		opt(&cfg)
	}

	handler := newHandler(cfg, newChecker(cfg))

	if !cfg.withManualStart {
		handler.Start()
	}

	return handler
}

// NewChecker creates a standalone health checker. If periodic checks have
// been configured (see WithPeriodicCheck), they will be started as well
// (if not explicitly turned off using WithManualStart).
// It operates in the same way as NewHandler but returning the Checker directly instead of the handler.
func NewChecker(options ...option) Checker {
	cfg := healthCheckConfig{
		statusCodeUp:   http.StatusOK,
		statusCodeDown: http.StatusServiceUnavailable,
		cacheTTL:       1 * time.Second,
		timeout:        30 * time.Second,
		maxErrMsgLen:   500,
		checks:         map[string]*Check{},
	}

	for _, opt := range options {
		opt(&cfg)
	}

	checker := newChecker(cfg)

	if !cfg.withManualStart {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
		defer cancel()

		checker.Start()
		checker.Check(ctx)
	}

	return checker
}

// WithMaxErrorMessageLength limits maximum number of characters
// in error messages.
func WithMaxErrorMessageLength(length uint) option {
	return func(cfg *healthCheckConfig) {
		cfg.maxErrMsgLen = length
	}
}

// WithDisabledDetails disables hides all data in the JSON response body but the the Status itself.
// Example: { "Status":"down" }
func WithDisabledDetails() option {
	return func(cfg *healthCheckConfig) {
		cfg.detailsDisabled = true
	}
}

// WithTimeout globally defines a timeout duration for all checks. You can still override
// this timeout by using the timeout value in the Check configuration.
// Default value is 30 seconds.
func WithTimeout(timeout time.Duration) option {
	return func(cfg *healthCheckConfig) {
		cfg.timeout = timeout
	}
}

// WithCustomStatusCodes allows to set custom HTTP Status code for the case when the system is evaluated to be
// up or down (based on check results).
// Default values are statusCodeUp = 200 (OK) and statusCodeDown = 503 (Service Unavailable).
func WithCustomStatusCodes(statusCodeUp int, statusCodeDown int) option {
	return func(cfg *healthCheckConfig) {
		cfg.statusCodeUp = statusCodeUp
		cfg.statusCodeDown = statusCodeDown
	}
}

// WithStatusListener registers a handler function that will be called whenever the overall system health
// Status changes. Attention: Ideally, this method should be quick and not block for too long.
func WithStatusListener(listener SystemStatusListener) option {
	return func(cfg *healthCheckConfig) {
		cfg.statusChangeListener = listener
	}
}

// WithManualStart prevents an automatic start of periodic checks (see NewHandler).
// If this configuration option is used and you want to start periodic checks yourself,
// you need to start them by using Start.
func WithManualStart() option {
	return func(cfg *healthCheckConfig) {
		cfg.withManualStart = true
	}
}

// WithDisabledCache disabled the check cache. This is not recommended in most cases.
// This will effectively lead to a health endpoint that initiates a new health check for each incoming HTTP request.
// This may have an impact on the systems that are being checked (especially if health checks are expensive).
// Caching also mitigates "denial of service" attacks.
func WithDisabledCache() option {
	return WithCacheDuration(0)
}

// WithCacheDuration sets the duration for how long the aggregated health check result will be
// cached. This is set to 1 second by default. Caching will prevent that each incoming HTTP request
// triggers a new health check. A duration of 0 will effectively disable the cache and has the same effect as
// WithDisabledCache.
func WithCacheDuration(duration time.Duration) option {
	return func(cfg *healthCheckConfig) {
		cfg.cacheTTL = duration
	}
}

// WithCheck adds a new health check that contributes to the overall service availability Status.
// This check will be triggered each time the health check HTTP endpoint is called (and the
// cache has expired, see WithCacheDuration). If health checks are expensive or
// you expect a lot of calls to the health endpoint, consider using WithPeriodicCheck instead.
func WithCheck(check Check) option {
	return func(cfg *healthCheckConfig) {
		cfg.checks[check.Name] = &check
	}
}

// WithPeriodicCheck adds a new health check that contributes to the overall service availability Status.
// The health check will be performed on a fixed schedule and will not be executed for each HTTP request
// (as in contrast to WithCheck). This allows to process a much higher number of HTTP requests without
// actually calling the checked services too often or to execute long running checks.
// The health endpoint always returns the last result of the periodic check.
// When periodic checks are started (happens automatically if WithManualStart is not used)
// they are also executed for the first time. Until all periodic checks have not been executed at least once,
// the overall availability Status will be "unknown" with HTTP Status code 503 (Service Unavailable).
func WithPeriodicCheck(refreshPeriod time.Duration, check Check) option {
	return func(cfg *healthCheckConfig) {
		check.updateInterval = refreshPeriod
		cfg.checks[check.Name] = &check
	}
}
