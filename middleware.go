package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Middleware allows to define a wrapper function for HTTP handlers. This allows to
// pre- and postprocess HTTP requests/responses before and/or after running health checks.
type Middleware func(next http.Handler) http.Handler

func newAuthMiddleware(sendStatusOnAuthFailure bool, authFunc func(r *http.Request) error) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := authFunc(r)
			if err != nil {
				if !sendStatusOnAuthFailure {
					http.Error(w, "Unauthorized", 401)
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(withAuthResult(r.Context(), err == nil)))
		})
	}
}

func newBasicAuthMiddleware(username string, password string, sendStatusOnAuthFailure bool) Middleware {
	return newAuthMiddleware(sendStatusOnAuthFailure, func(r *http.Request) error {
		user, pass, _ := r.BasicAuth()
		if user != username || pass != password {
			return fmt.Errorf("authentication failed")
		}
		return nil
	})
}

func newTimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
