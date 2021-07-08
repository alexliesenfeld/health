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
		// This function must return an error if the checked service is considered not available.
		// Check is a required attribute.
		Check func(ctx context.Context) error
		// Timeout will override the global timeout value, if it is smaller than the global timeout (see WithTimeout).
		Timeout time.Duration
		// FailureTolerance will set a duration for how long a service must be "in error" until it
		// is considered unavailable.
		FailureTolerance time.Duration
		// FailureToleranceThreshold will set a maximum number of consecutive check fails until the service
		// is considered unavailable.
		FailureToleranceThreshold uint
		refreshInterval           time.Duration
	}

	option func(*healthCheckConfig)
)

// New creates a new health check http.Handler. If periodic checks have
// been configured (see WithPeriodicCheck), they will be started as well
// (if not explicitly turned off using WithManualPeriodicCheckStart).
func New(options ...option) http.Handler {
	cfg := healthCheckConfig{
		cacheDuration: 1 * time.Second,
		timeout:       30 * time.Second,
		maxErrMsgLen:  500,
	}

	for _, option := range options {
		option(&cfg)
	}

	return newHandler(cfg.middleware, newChecker(cfg))
}

// WithMaxErrorMessageLength limits maximum number of characters
// in error messages.
func WithMaxErrorMessageLength(length uint) option {
	return func(cfg *healthCheckConfig) {
		cfg.maxErrMsgLen = length
	}
}

// WithMiddleware allows to add a Middleware to the processing chain of HTTP requests.
// Middleware is a wrapper for HTTP handlers. This allows to pre- and postprocess HTTP
// requests/responses before and/or after running the health checks.
func WithMiddleware(mw Middleware) option {
	return func(cfg *healthCheckConfig) {
		cfg.middleware = append(cfg.middleware, mw)
	}
}

// WithCustomAuth adds a custom authentication middleware so you can use you own authentication
// functionality to integrate with this library. Parameter sendStatusOnAuthFailure=true
// allows to return the HTTP status code and the aggregated overall status without the details
// even if authentication was not successful. This is useful to allow publicly available health checks
// that do not expose system details.
// Example: If authentication is not successful and sendStatusOnAuthFailure=true, then HTTP status
// code 200 (OK) will be returned in case the service is up or 503 (Service Unavailable) if the
// service is considered down. Additionally, the response body will only contain an aggregated status
// but without any details (e.g. { "status" : "UP" }). In case sendStatusOnAuthFailure=false
// and authentication fails, then an empty response body will be returned along with HTTP status code 401
// (Unauthorized).
func WithCustomAuth(sendStatusOnAuthFailure bool, authFunc func(r *http.Request) error) option {
	return WithMiddleware(newAuthMiddleware(sendStatusOnAuthFailure, authFunc))
}

// WithBasicAuth adds a basic authentication middleware. Parameter sendStatusOnAuthFailure=true
// allows to pass the HTTP status code to the user even if authentication was not successful.
// This is useful to allow publicly available health checks that do not expose system details.
// Example: If authentication is not successful and sendStatusOnAuthFailure=true, then HTTP status
// code 200 (OK) will be returned in case the service is up or 503 (Service Unavailable) if the
// service is considered down. Additionally, the response body will only contain an aggregated status
// but without any details (e.g. { "status" : "UP" }). In case sendStatusOnAuthFailure=false
// and authentication fails, then an empty response body will be returned along with HTTP status code 401
// (Unauthorized).
func WithBasicAuth(username string, password string, sendStatusOnAuthFailure bool) option {
	return WithMiddleware(newBasicAuthMiddleware(username, password, sendStatusOnAuthFailure))
}

// WithTimeout globally defines a timeout duration for all checks. You can still override
// this timeout by using the timeout value in the Check configuration.
// Default value is 30 seconds.
func WithTimeout(timeout time.Duration) option {
	return WithMiddleware(newTimeoutMiddleware(timeout))
}

// WithManualPeriodicCheckStart prevents an automatic start of periodic checks (see New).
// If this configuation option is used and you want to start periodic checks yourself,
// you need to start them by using StartPeriodicChecks.
func WithManualPeriodicCheckStart() option {
	return func(cfg *healthCheckConfig) {
		cfg.manualPeriodicCheckStart = true
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
		cfg.cacheDuration = duration
	}
}

// WithCheck adds a new health check that contributes to the overall service availability status.
// This check will be triggered each time the health check HTTP endpoint is called (and the
// cache has expired, see WithCacheDuration). If health checks are expensive or
// you expect a lot of calls to the health endpoint, consider using WithPeriodicCheck instead.
func WithCheck(check Check) option {
	return func(cfg *healthCheckConfig) {
		cfg.checks = append(cfg.checks, &check)
	}
}

// WithPeriodicCheck adds a new health check that contributes to the overall service availability status.
// The health check will be performed on a fixed schedule and will not be executed for each HTTP request
// (as in contrast to WithCheck). This allows to process a much higher number of HTTP requests without
// actually calling the checked services too often or to execute long running checks.
// The health endpoint always returns the last result of the periodic check.
// When periodic checks are started (happens automatically if WithManualPeriodicCheckStart is not used)
// they are also executed for the first time. Until all periodic checks have not been executed at least once,
// the overall availability status will be "UNKNOWN" with HTTP status code 503 (Service Unavailable).
func WithPeriodicCheck(refreshPeriod time.Duration, check Check) option {
	return func(cfg *healthCheckConfig) {
		check.refreshInterval = refreshPeriod
		cfg.checks = append(cfg.checks, &check)
	}
}
