package main

import (
	"context"
	"github.com/InVisionApp/go-health/checkers"
	"github.com/alexliesenfeld/health"
	"log"
	"net/http"
	"net/url"
	"time"
)

func main() {
	// Create checkers as usual.
	googleURL, err := url.Parse("https://www.google.com")
	if err != nil {
		log.Fatalln(err)
	}

	check, err := checkers.NewHTTP(&checkers.HTTPConfig{
		URL:     googleURL,
		Timeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Add the check in the Checker configuration
	http.Handle("/health", health.NewHandler(
		health.NewChecker(
			health.WithCheck(health.Check{
				Name: "google",
				Check: func(_ context.Context) error {
					_, err := check.Status()
					return err
				},
				// InVisionApp/go-health does not support context based timeouts.
				// This timeout of 5 seconds will not propagate to the check function,
				// because check.Status (see above) does not accept a context.
				// Timeouts need to be set before the check is used (see usage
				// of checkers.NewHTTP above, where we set a timeout of 3 seconds).
				// Since this value here is set to a high number by default
				// (currently 10 seconds), you can basically leave it away here and only
				// define the timeout in the checkers.HTTPConfig above.
				Timeout: 5 * time.Second,
			}),
		)))
	log.Fatalln(http.ListenAndServe(":3000", nil))
}
