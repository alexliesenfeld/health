package middleware

import (
	"github.com/alexliesenfeld/health"
	"log"
	"net/http"
	"time"
)

// BasicLogger is a basic logger that is mostly used to showcase this library.
func BasicLogger() health.Middleware {
	return func(next health.MiddlewareFunc) health.MiddlewareFunc {
		return func(r *http.Request) health.CheckerResult {
			now := time.Now()
			result := next(r)
			log.Printf("processed health check request in %f seconds (result: %s)",
				time.Now().Sub(now).Seconds(), result.Status)
			return result
		}
	}
}
