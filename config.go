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
		Name string // Required

		// Check is the check function that will be executed to check availability.
		// This function must return an error if the checked service is considered
		// not available. Check is a required attribute.
		Check func(ctx context.Context) error // Required

		// Timeout will override the global timeout value, if it is smaller than
		// the global timeout (see WithTimeout).
		Timeout time.Duration // Optional

		// MaxTimeInError will set a duration for how long a service must be
		// in an error state until it is considered down/unavailable.
		MaxTimeInError time.Duration // Optional

		// MaxContiguousFails will set a maximum number of contiguous
		// check fails until the service is considered down/unavailable.
		MaxContiguousFails uint // Optional

		// StatusListener allows to set a listener that will be called
		// whenever the AvailabilityStatus (e.g. from "up" to "down").
		// The function is allowed to add add values to the context in
		// parameter ctx. The new context is expected in the return value
		// of the function. The new context will be used for the rest of the
		// check execution (i.e, will be passed to all downstream function
		// calls, such as all following lifecycle functions).
		StatusListener func(ctx context.Context, state CheckState) context.Context // Optional

		// BeforeCheckListener is a callback function that will be called
		// right before a components availability status is checked.
		// The function is allowed to add add values to the context in
		// parameter ctx. The new context is expected in the return value
		// of the function. The new context will be used for the rest of the
		// check execution (i.e, will be passed to all downstream function
		// calls, such as all following lifecycle functions).
		BeforeCheckListener func(ctx context.Context, name string, state CheckState) context.Context // Optional

		// BeforeCheckListener is a callback function that will be called
		// right after a components availability status has been checked.
		AfterCheckListener func(ctx context.Context, state CheckState)

		updateInterval time.Duration
	}

	option func(*healthCheckConfig)
)

// NewChecker creates a new Checker. The provided options will be
// used to modify its configuration.
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

	return newDefaultChecker(cfg)
}

// WithMaxErrorMessageLength limits maximum number of characters
// in error messages. Default is 500.
func WithMaxErrorMessageLength(length uint) option {
	return func(cfg *healthCheckConfig) {
		cfg.maxErrMsgLen = length
	}
}

// WithDisabledDetails disables all data in the JSON response body. The AvailabilityStatus will be the only
// content. Example: { "status":"down" }. Enabled by default.
func WithDisabledDetails() option {
	return func(cfg *healthCheckConfig) {
		cfg.detailsDisabled = true
	}
}

// WithTimeout defines a timeout duration for all checks. You can override
// this timeout by using the timeout value in the Check configuration.
// Default value is 30 seconds.
func WithTimeout(timeout time.Duration) option {
	return func(cfg *healthCheckConfig) {
		cfg.timeout = timeout
	}
}

// WithStatusListener registers a handler function that will be called whenever the overall/aggregated system health
// status changes (e.g. from "up" to "down").
func WithStatusListener(listener func(context.Context, AvailabilityStatus, map[string]CheckState) context.Context) option {
	return func(cfg *healthCheckConfig) {
		cfg.statusChangeListener = listener
	}
}

// WithBeforeCheckListener adds a listener function that is triggered
// before Checker.Check is executed. The function is allowed to add add
// values to the context in parameter ctx. The new context is expected
// in the return value of the function. The new context will be used
// for the rest of the check execution (i.e, will be passed to all
// downstream function calls, such as all following lifecycle functions).
// Attention: This function will not be executed for periodic checks!
// This is because periodic check functions are executed (asynchronously)
// on their own time schedule, separate from regular (synchronous)
// checking that happens when by invoking Checker.Check.
// What it means for you is that this function will be called for non-periodic
// checks on each HTTP request, but never for periodic checks.
func WithBeforeCheckListener(listener func(context.Context, AvailabilityStatus, map[string]CheckState) context.Context) option {
	return func(cfg *healthCheckConfig) {
		cfg.beforeSystemCheckListener = listener
	}
}

// WithAfterCheckListener adds a listener function that is triggered
// after Checker.Check was executed.
// Attention: This function will not be executed for periodic checks!
// This is because periodic check functions are executed (asynchronously)
// on their own time schedule, separate from regular (synchronous)
// checking that happens when by invoking Checker.Check.
// What it means for you is that this function will be called for non-periodic
// checks on each HTTP request, but never for periodic checks.
func WithAfterCheckListener(listener func(context.Context, AvailabilityStatus, map[string]CheckState)) option {
	return func(cfg *healthCheckConfig) {
		cfg.afterSystemCheckListener = listener
	}
}

// WithDisabledCache disabled the check cache. This is not recommended in most cases.
// This will effectively lead to a health endpoint that initiates a new health check for each incoming HTTP request.
// This may have an impact on the systems that are being checked (especially if health checks are expensive).
// Caching also mitigates "denial of service" attacks. Caching is enabled by default.
func WithDisabledCache() option {
	return WithCacheDuration(0)
}

// WithCacheDuration sets the duration for how long the aggregated health check result will be
// cached. By default, the cache TTL (i.e, the duration for how long responses will be cached) is set to 1 second.
// Caching will prevent that each incoming HTTP request triggers a new health check. A duration of 0 will
// effectively disable the cache and has the same effect as WithDisabledCache.
func WithCacheDuration(duration time.Duration) option {
	return func(cfg *healthCheckConfig) {
		cfg.cacheTTL = duration
	}
}

// WithCheck adds a new health check that contributes to the overall service availability status.
// This check will be triggered each time Checker.Check is called (i.e., for each HTTP request).
// If health checks are expensive or you expect a bigger amount of requests on your the health endpoint,
// consider using WithPeriodicCheck instead.
func WithCheck(check Check) option {
	return func(cfg *healthCheckConfig) {
		cfg.checks[check.Name] = &check
	}
}

// WithPeriodicCheck adds a new health check that contributes to the overall service availability status.
// The health check will be performed on a fixed schedule and will not be executed for each HTTP request
// (as in contrast to WithCheck). This allows to process a much higher number of HTTP requests without
// actually calling the checked services too often or to execute long running checks.
// This way Checker.Check (and the health endpoint) always returns the last result of the periodic check.
func WithPeriodicCheck(refreshPeriod time.Duration, check Check) option {
	return func(cfg *healthCheckConfig) {
		check.updateInterval = refreshPeriod
		cfg.checks[check.Name] = &check
	}
}
