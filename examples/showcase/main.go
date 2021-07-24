package main

import (
	"context"
	"fmt"
	"github.com/alexliesenfeld/health"
	"github.com/alexliesenfeld/health/interceptors"
	"github.com/alexliesenfeld/health/middleware"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// This example shows all core features of this library. Please note that this example is not to show how a
// real-world health check implementation would look like but merely to give you an idea of what you can
// achieve with it.
func main() {
	// Create a new Checker
	checker := health.NewChecker(

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10),

		// Set for how long check responses are cached.
		health.WithDisabledCache(),
		health.WithCacheDuration(2*time.Second),

		// Cut error message length
		health.WithMaxErrorMessageLength(500),

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// Disable the details in the health check results.
		// This will configure the checker to only return the aggregated
		// health status but no component details.
		health.WithDisabledDetails(),

		// Disable automatic Checker start. By default, the Checker is started automatically.
		// This configuration option disables this behaviour, so you can delay startup. You need to start
		// the Checker explicitly though see Checker.Start).
		health.WithDisabledAutostart(),

		// This listener will be called whenever system health status changes (e.g., from "up" to "down").
		health.WithStatusListener(onSystemStatusChanged),

		// A simple successFunc to see if a fake database connection is up.
		health.WithCheck(health.Check{
			// Each check gets a unique name
			Name: "database",
			// The check function. Return an error if the service is unavailable.
			Check: func(ctx context.Context) error {
				return nil // no error
			},
			// A successFunc specific timeout.
			Timeout: 2 * time.Second,
			// A status listener that will be called if status of this component changes.
			StatusListener: onComponentStatusChanged,
			// An interceptor pre- and post-processes each call to the check function
			Interceptors: []health.Interceptor{interceptors.BasicLogger()},
			// The check is allowed to fail up to 5 times in a row
			// until considered unavailable.
			MaxContiguousFails: 5,
			// Check is allowed to stay for up to 1 minute in an error
			// state until considered unavailable.
			MaxTimeInError: 1 * time.Minute,
		}),

		// The following check will be executed periodically every 10 seconds with an initial delay of 3 seconds.
		health.WithPeriodicCheck(10*time.Second, 3*time.Second, health.Check{
			// Each check gets a unique name
			Name: "search-engine",
			// The check function. Return an error if the service is unavailable.
			Check: volatileFunc(),
			// A successFunc specific timeout.
			Timeout: 2 * time.Second,
			// A status listener that will be called if status of this component changes.
			StatusListener: onComponentStatusChanged,
			// An interceptor pre- and post-processes each call to the check function
			Interceptors: []health.Interceptor{interceptors.BasicLogger()},
			// The check is allowed to fail up to 5 times in a row
			// until considered unavailable.
			MaxContiguousFails: 5,
			// Check is allowed to stay for up to 1 minute in an error
			// state until considered unavailable.
			MaxTimeInError: 1 * time.Minute,
		}),
	)

	// OPTIONAL: This is only required because we used WithDisabledAutostart option above.
	// The Checker is usually automatically started (see NewChecker), so starting it explicitly
	// should almost never be required.
	checker.Start()

	// Create a new handler that is able to process HTTP requests to a health endpoint.
	handler := health.NewHandler(checker,

		// A result writer writes a check result into an HTTP response.
		// JSONResultWriter is used by default.
		health.WithResultWriter(health.NewJSONResultWriter()),

		// A list of middlewares to pre- and post-process health check requests.
		health.WithMiddleware(
			middleware.BasicLogger(),                 // This middleware will log incoming requests
			middleware.BasicAuth("user", "password"), // Removes check details based on basic auth
		),

		// Set a custom HTTP status code that should be used if the system is considered "up".
		health.WithStatusCodeUp(200),

		// Set a custom HTTP status code that should be used if the system is considered "down".
		health.WithStatusCodeDown(503),
	)

	// We Create a new http.Handler that provides health successFunc information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", handler)
	http.ListenAndServe(":3000", nil)
}

func onComponentStatusChanged(_ context.Context, name string, state health.CheckState) {
	log.Println(fmt.Sprintf("component %s changed status to %s", name, state.Status))
}

func onSystemStatusChanged(_ context.Context, state health.CheckerState) {
	log.Println(fmt.Sprintf("system status changed to %s", state.Status))
}

func volatileFunc() func(ctx context.Context) error {
	var count uint32
	return func(ctx context.Context) error {
		defer atomic.AddUint32(&count, 1)
		if count%2 == 0 {
			return fmt.Errorf("this is a check error") // example error
		}
		return nil
	}
}
