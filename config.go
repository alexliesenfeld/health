package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type (
	// Middleware allows to define a wrapper function for HTTP handlers
	// (sometimes referred to as "middleware" functions). This allows to
	// preprocess and postprocess HTTP requests/responses before or after
	// running the health checks.
	Middleware func(next http.Handler) http.Handler

	// Check allows to define all aspects of health checks.
	Check struct {
		// The Name must be unique among all checks. This is a required attribute.
		Name string
		// Check is the check function that will be excuted to check availability status.
		// This function must return an error if the service it is checking is cosidered
		// not available. This is a required attribute.
		Check func(ctx context.Context) error
		// Timeout will override the global timeout value, if it is smaller than the global timeout.
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

	ckr := newChecker(cfg)

	if !cfg.manualPeriodicCheckStart {
		ckr.StartPeriodicChecks()
	}

	return newHandler(cfg.middleware, ckr)
}

// WithMaxErrorMessageLength limits maximum number of characters
// in error messages.
func WithMaxErrorMessageLength(length uint) option {
	return func(cfg *healthCheckConfig) {
		cfg.maxErrMsgLen = length
	}
}

// WithMiddlewareHandler allows to add a Middleware to the processing chain of HTTP requests.
// Middleware is a wrapper for HTTP handlers (sometimes referred to as
// "middleware" functions). This allows to preprocess and postprocess HTTP
// requests/responses before or after running the health checks.
func WithMiddlewareHandler(mw Middleware) option {
	return func(cfg *healthCheckConfig) {
		cfg.middleware = append(cfg.middleware, mw)
	}
}

// WithMiddleware allows to add a Middleware to the processing chain of HTTP requests.
// Middleware is a wrapper function for HTTP handlers (sometimes referred to as
// "middleware" functions). This allows to preprocess and postprocess HTTP
// requests/responses before or after running the health checks.
func WithMiddleware(hf http.HandlerFunc) option {
	return WithMiddlewareHandler(func(h http.Handler) http.Handler {
		return http.HandlerFunc(hf)
	})
}

// WithCustomAuth adds a custom authentication middleware so you can use you own authentication
// functionality to integrate with this library. Parameter sendStatusOnAuthFailure=true
// allows to pass the HTTP status code to the user even if authentication was not successful.
// This is useful to allow publicly available health checks that do not expose system details.
// Example: If authentication is not successful and sendStatusOnAuthFailure=true, then HTTP status
// code 200 (OK) will be returned in case the service is up or 503 (Service Unavailable) if the
// service is considered down. On the other hand, if sendStatusOnAuthFailure=false,
// HTTP status code 401 (Unauthorized) will always be returned if authentication fails.
func WithCustomAuth(sendStatusOnAuthFailure bool, authFunc func(r *http.Request) error) option {
	return WithMiddlewareHandler(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := authFunc(r)
			if err != nil {
				if !sendStatusOnAuthFailure {
					http.Error(w, "Unauthorized", 401)
					return
				}
				r = r.WithContext(withAuthResult(r.Context(), false))
			}
			next.ServeHTTP(w, r.WithContext(withAuthResult(r.Context(), err == nil)))
		})
	})
}

// WithBasicAuth adds a basic authentication middleware. Parameter sendStatusOnAuthFailure=true
// allows to pass the HTTP status code to the user even if authentication was not successful.
// This is useful to allow publicly available health checks that do not expose system details.
// Example: If authentication is not successful and sendStatusOnAuthFailure=true, then HTTP
// status code 200 (OK) will be returned in case the service is up or 503 (Service Unavailable)
// if the service is cosidered down. On the other hand, if sendStatusOnAuthFailure=false,
// HTTP status code 401 (Unauthorized) will always be returned if authentication fails.
func WithBasicAuth(username string, password string, sendStatusOnAuthFailure bool) option {
	return WithCustomAuth(sendStatusOnAuthFailure, func(r *http.Request) error {
		user, pass, _ := r.BasicAuth()
		if user != username || pass != password {
			return fmt.Errorf("authentication failed")
		}
		return nil
	})
}

// WithTimeout defines a timeout duration for all checks. You can still override
// this timeout by using the timeout value in the Check configuration.
// Default value is 30 seconds.
func WithTimeout(timeout time.Duration) option {
	return WithMiddlewareHandler(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	})
}

// WithManualPeriodicCheckStart prevents an automatic start of periodic checks (see New).
// If you want periodic checks to start, you need to start them by using
// StartPeriodicChecks manually.
func WithManualPeriodicCheckStart() option {
	return func(cfg *healthCheckConfig) {
		cfg.manualPeriodicCheckStart = true
	}
}

// WithDisabledCache disabled the check cache. This is not recommended in most cases
// (e.g. for all publicly available health checks). This will trigger a new health check
// for each incoming HTTP request, which might have an impact on the systems that are
// being checked (especially if health checks functions are expected to be expensive).
// Caching is considered to contribute to DDoS attack prevention, which calling this
// function effectively disables.
func WithDisabledCache() option {
	return WithCacheDuration(0)
}

// WithCacheDuration sets the duration for how long the aggregated health check result will be
// cached. This is set to 1 second by default. Caching will prevent that each HTTP request
// triggers a new health check. This is especially useful when checks are considered expensive.
// A duration of 0 will effectively disable the cache and has the same effect as
// WithDisabledCache.
func WithCacheDuration(duration time.Duration) option {
	return func(cfg *healthCheckConfig) {
		cfg.cacheDuration = duration
	}
}

// WithCheck adds a new health check that contributes to the overall service availability status.
// This check will be triggered each time the health check HTTP endpoint is called (and the
// cache has expired, see WithCacheDuration). If health checks are considered expensive or
// you expect a lot of calls to the health endpoint, consider using WithPeriodicCheck instead.
func WithCheck(check Check) option {
	return func(cfg *healthCheckConfig) {
		cfg.checks = append(cfg.checks, &check)
	}
}

// WithPeriodicCheck adds a new health check that contributes to the overall service availability status.
// The health check will be performed on a fixed schedule and will not be executed for each HTTP request
// any more. The result in between checks is cached and provided as a result for HTTP requests to
// the health HTTP endpoint. This allows to process a much higher number of HTTP requests without actually
// calling the checked services too often.
func WithPeriodicCheck(refreshPeriod time.Duration, check Check) option {
	return func(cfg *healthCheckConfig) {
		check.refreshInterval = refreshPeriod
		cfg.checks = append(cfg.checks, &check)
	}
}
