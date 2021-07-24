package main

import (
	"context"
	"fmt"
	"github.com/alexliesenfeld/health"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// This example shows how to add check interceptors and handler middleware for pre- and post-processing.
// Both, interceptors and middleware allow to create re-usable functions, mostly used for cross-cutting
// functionality, such as logging, tracing, authentication, caching, etc.
func main() {

	// Create a new Checker
	checker := health.NewChecker(
		// A simple successFunc to see if a fake file system up.
		health.WithCheck(health.Check{
			Name:         "filesystem",
			Timeout:      2 * time.Second, // A successFunc specific timeout.
			Interceptors: []health.Interceptor{createCheckLogger, logCheck},
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this is a check error") // example error
			},
		}),
	)

	handler := health.NewHandler(checker, health.WithMiddleware(createRequestLogger, logRequest))

	// We Create a new http.Handler that provides health successFunc information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", handler)
	http.ListenAndServe(":3000", nil)
}

func createCheckLogger(next health.InterceptorFunc) health.InterceptorFunc {
	return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
		logger := getLogger(ctx)
		if logger == nil {
			logger = log.NewEntry(log.New())
		}
		logger = logger.WithFields(log.Fields{"name": name})
		ctx = setLogger(ctx, logger)
		return next(ctx, name, state)
	}
}

func logCheck(next health.InterceptorFunc) health.InterceptorFunc {
	return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
		logger := getLogger(ctx)
		logger.Infof("starting component check")
		res := next(ctx, name, state)
		logger.Infof("component check finished")
		return res
	}
}

func createRequestLogger(next health.MiddlewareFunc) health.MiddlewareFunc {
	return func(r *http.Request) health.CheckerResult {
		logger := getLogger(r.Context())
		if logger == nil {
			logger = log.WithFields(log.Fields{"request": uuid.New()})
		}
		ctx := setLogger(r.Context(), logger)
		return next(r.WithContext(ctx))
	}
}

func logRequest(next health.MiddlewareFunc) health.MiddlewareFunc {
	return func(r *http.Request) health.CheckerResult {
		logger := getLogger(r.Context())
		logger.Infof("starting to process health check request")
		res := next(r)
		logger.Infof("finished processing of health check request")
		return res
	}
}

func setLogger(ctx context.Context, logger *log.Entry) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

func getLogger(ctx context.Context) *log.Entry {
	logger, ok := ctx.Value("logger").(*log.Entry)
	if ok {
		return logger
	}
	return nil
}
