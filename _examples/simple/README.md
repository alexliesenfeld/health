```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alexliesenfeld/health"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"time"
)

func main() {
	db, _ := sql.Open("sqlite3", "simple.sqlite")
	defer db.Close()

	// Create a new Checker
	checker := health.NewChecker(

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// A simple check to see if database connection is up.
		health.WithCheck(health.Check{
			Name:    "database",
			Timeout: 2 * time.Second, // A a check specific timeout.
			Check:   db.PingContext,
		}),

		// The following check will be executed periodically every 30 seconds.
		health.WithPeriodicCheck(30*time.Second, health.Check{
			Name: "search",
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this makes the check fail")
			},
		}),
	)

	// This will start periodic checks and do a first health check
	// to test for an initial system status.
	checker.Start()

	// We Create a new http.Handler that provides health check information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))
	http.ListenAndServe(":3000", nil)
}
```