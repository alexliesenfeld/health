package main

import (
	"github.com/alexliesenfeld/health"
	httpCheck "github.com/hellofresh/health-go/v4/checks/http"
	"log"
	"net/http"
)

func main() {
	http.Handle("/health", health.NewHandler(
		health.NewChecker(
			health.WithCheck(health.Check{
				Name: "google",
				Check: httpCheck.New(httpCheck.Config{
					URL: "https://www.google.com",
				}),
			}),
		)))
	log.Fatalln(http.ListenAndServe(":3000", nil))
}
