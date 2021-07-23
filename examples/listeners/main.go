package main

import (
	"context"
	"fmt"
	"github.com/alexliesenfeld/health"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync/atomic"
	"time"
)

func main() {
	// Create a new Checker
	checker := health.NewChecker(

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// A simple successFunc to see if a fake database connection is up.
		health.WithCheck(health.Check{
			Name:           "database",
			Timeout:        2 * time.Second, // A successFunc specific timeout.
			StatusListener: onComponentStatusChanged,
			Check: func(ctx context.Context) error {
				return nil // no error
			},
		}),

		// A simple successFunc to see if a fake file system up.
		health.WithCheck(health.Check{
			Name:           "filesystem",
			Timeout:        2 * time.Second, // A successFunc specific timeout.
			StatusListener: onComponentStatusChanged,
			Check: func(ctx context.Context) error {
				return nil // example error
			},
		}),

		// The following check will be executed periodically every 30 seconds.
		health.WithPeriodicCheck(5*time.Second, 10*time.Second, health.Check{
			Name:           "search-engine",
			StatusListener: onComponentStatusChanged,
			Check:          volatileFunc(),
		}),

		// This listener will be called whenever system health status changes (e.g., from "up" to "down").
		health.WithStatusListener(onSystemStatusChanged),
	)

	// We Create a new http.Handler that provides health successFunc information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))
	http.ListenAndServe(":3000", nil)
}

func onComponentStatusChanged(_ context.Context, name string, state health.CheckState) {
	log.Infof("component %s changed status to %s", name, state.Status)
}

func onSystemStatusChanged(_ context.Context, state health.CheckerState) {
	log.Infof("system status changed to %s", state.Status)
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
