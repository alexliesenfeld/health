package main

import (
	"context"
	"fmt"
	"github.com/alexliesenfeld/health"
	"log"
	"net/http"
	"time"
)

func main() {

	// Create a new Checker
	checker := health.NewChecker(

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// A simple check to see if database connection is up.
		health.WithCheck(health.Check{
			Name:    "database",
			Timeout: 2 * time.Second, // A check specific timeout.
			Check: func(ctx context.Context) error {
				return nil
			},
		}),

		// The following check will be executed periodically every 30 seconds.
		health.WithPeriodicCheck(15*time.Second, health.Check{
			Name:                "search",
			MaxConsecutiveFails: 10,
			MaxTimeInError:      1 * time.Minute,
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this makes the check fail")
			},
		}),

		health.WithStatusListener(func(status health.AvailabilityStatus, state map[string]health.CheckState) {
			log.Println(fmt.Sprintf("system health changed to status %s", status))
		}),
	)

	checker.Start()

	// Create a new http.Handler that provides health check information.
	http.Handle("/healthcheck", health.NewHandler(checker))
	http.ListenAndServe(":3000", nil)
}
