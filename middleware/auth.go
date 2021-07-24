package middleware

import (
	"github.com/alexliesenfeld/health"
	"net/http"
)

// BasicAuth is a middleware that removes check details (such as service names, error messages, etc.) from the
// HTTP response on authentication failure. Authentication is performed based on basic access authentication
// (https://en.wikipedia.org/wiki/Basic_access_authentication).
//
// This is useful if you want to allow the aggregated
// result to be visible to all clients, but provide details only to fully authenticated senders.
//
// Attention: To be able to prevent access altogether if authentication fails, consider using an
// HTTP basic auth middleware instead. You can can easily use most such middleware implementations with the
// Handler (e.g., https://github.com/99designs/basicauth-go). This libraries middleware (health.Middleware)
// is only for pre- and post-processing results but not to deal with the HTTP request and response objects.
func BasicAuth(username, password string) health.Middleware {
	return CustomAuth(func(r *http.Request) bool {
		reqUser, reqPassword, ok := r.BasicAuth()
		return ok && username == reqUser && password == reqPassword
	})
}

// CustomAuth is a middleware that removes check details (such as service names, error messages, etc.) from the
// HTTP response on authentication failure. To find out if authentication was successful, the provided function will be
// executed (provided in argument 'authFunc').
//
// This middleware is useful if you want to allow the aggregated result to be visible to all clients, but provide
// details only to fully authenticated senders.
//
// Attention: To be able to prevent access altogether if authentication fails, consider using an
// HTTP basic auth middleware instead. You can can easily use most such middleware implementations with the
// Handler (e.g., https://github.com/99designs/basicauth-go). This libraries middleware (health.Middleware)
// is only for pre- and post-processing results but not to deal with the HTTP request and response objects.
func CustomAuth(authFunc func(r *http.Request) bool) health.Middleware {
	return func(next health.MiddlewareFunc) health.MiddlewareFunc {
		return func(r *http.Request) health.CheckerResult {
			authSuccess := authFunc(r)
			result := next(r)
			if !authSuccess {
				result.Details = nil
			}
			return result
		}
	}
}
