package interceptors

import (
	"context"
	"github.com/alexliesenfeld/health"
	"log"
	"time"
)

// BasicLogger is a basic logger that is mostly used to showcase this library.
func BasicLogger() health.Interceptor {
	return func(next health.InterceptorFunc) health.InterceptorFunc {
		return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
			now := time.Now()
			result := next(ctx, name, state)
			log.Printf("executed health check function of component %s in %f seconds (result: %s)",
				name, time.Since(now).Seconds(), result.Status)
			return result
		}
	}
}
