package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alexliesenfeld/health"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"time"
)

// This is a an example configuration for Kubernetes liveness and readiness checks.
// Please note that Kubernetes readiness and especially liveness checks need to be designed
// with care to not cause any unintended behaviour (such as unexpected pod restarts, cascading failures, etc.).
// Please refer to the following articles for guidance:
// - https://www.innoq.com/en/blog/kubernetes-probes/
// - https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-how-to-avoid-shooting-yourself-in-the-foot/
// - https://srcco.de/posts/kubernetes-liveness-probes-are-dangerous.html
func main() {
	db, _ := sql.Open("sqlite3", "simple.sqlite")
	defer db.Close()

	// Create a new Checker for our readiness check.
	readinessCheck := health.NewChecker(

		// Configure a global timeout that will be applied to all check functions.
		health.WithTimeout(10*time.Second),

		// A check configuration to see if our database connection is up.
		// Be wary though that this should be a "service private" database instance.
		// If many of your services use the same database instance, the readiness checks
		// of all of these services will start failing on every small database hick-up.
		// This is most likely not what you want. For guidance, please refer to the links
		// listed in the main function documentation above.
		health.WithCheck(health.Check{
			Name:  "database", // A unique check name.
			Check: db.PingContext,
		}),

		// The following check will be executed periodically every 15 seconds
		// started with an initial delay of 3 seconds. The check function will NOT
		// be executed for each HTTP request.
		health.WithPeriodicCheck(15*time.Second, 3*time.Second, health.Check{
			Name: "search",
			// The check function checks the health of a component. If an error is
			// returned, the component is considered unavailable ("down").
			// The context contains a deadline according to the configuration of
			// the Checker (global and .
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this makes the check fail")
			},
		}),

		// Set a status listener that will be invoked when the health status changes.
		// More powerful hooks are also available (see docs).
		health.WithStatusListener(func(ctx context.Context, state health.CheckerState) {
			log.Println(fmt.Sprintf("health status changed to %s", state.Status))
		}),
	)

	// Liveness check should only contain checks that identify if the service is locked up and cannot
	// recover (deadlocks, etc.). It should just respond with 200 OK in most cases.
	// Reason:
	livenessCheck := health.NewChecker()

	// Create a new health check http.Handler that returns the health status
	// serialized as a JSON string. You can pass pass further configuration
	// options to NewHandler to modify default configuration.
	http.Handle("/live", health.NewHandler(livenessCheck))
	http.Handle("/ready", health.NewHandler(readinessCheck))

	log.Fatalln(http.ListenAndServe(":3000", nil))
}
