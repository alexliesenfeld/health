package main

import (
	"database/sql"
	"github.com/alexliesenfeld/health"
	httpCheck "github.com/hellofresh/health-go/v4/checks/http"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
)

// This is a very simple example that shows the basic features of this library.
func main() {
	db, _ := sql.Open("sqlite3", "simple.sqlite")
	defer db.Close()

	// Create a new Checker
	checker := health.NewChecker(
		// A simple check to see if database connection is up.
		health.WithCheck(health.Check{
			Name: "google",
			Check: httpCheck.New(httpCheck.Config{
				URL: "https://www.google.com",
			}),
		}),
	)

	// We Create a new http.Handler that provides health check information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))
	log.Println(http.ListenAndServe(":3000", nil))
}
