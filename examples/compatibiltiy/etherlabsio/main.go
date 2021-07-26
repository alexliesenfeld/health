package main

import (
	"github.com/alexliesenfeld/health"
	"github.com/etherlabsio/healthcheck/v2/checkers"
	"log"
	"net/http"
)

func main() {
	http.Handle("/health", health.NewHandler(
		health.NewChecker(
			health.WithCheck(health.Check{
				Name:  "disk",
				Check: checkers.DiskSpace("/var/log", 90).Check,
			}),
		)))
	log.Fatalln(http.ListenAndServe(":3000", nil))
}
