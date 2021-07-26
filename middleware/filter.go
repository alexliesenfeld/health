package middleware

import (
	"github.com/alexliesenfeld/health"
	"net/http"
)

// FullDetailsOnQueryParam is a middleware that removes check details (such as service names, error messages, etc.)
// from the HTTP response unless the request contained a query parameter named like argument 'queryParamName'. If
// a query parameter is not present in the HTTP request, the response will only contain the aggregated health status.
func FullDetailsOnQueryParam(queryParamName string) health.Middleware {
	return func(next health.MiddlewareFunc) health.MiddlewareFunc {
		return func(r *http.Request) health.CheckerResult {
			_, fullDetails := r.URL.Query()[queryParamName]
			result := next(r)
			if !fullDetails {
				result.Details = nil
			}
			return result
		}
	}
}
