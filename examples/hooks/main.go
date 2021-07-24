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
			Interceptors: []health.Interceptor{createLogger, logComponentCheck, logStatusChange},
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this is a check error") // example error
			},
		}),

		health.WithStatusListener(logSystemStatusChange),
	)

	handler := health.NewHandler(checker, health.WithMiddleware(createSystemLogger, logCheck))

	// We Create a new http.Handler that provides health successFunc information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", handler)
	http.ListenAndServe(":3000", nil)
}

func createLogger(next health.InterceptorFunc) health.InterceptorFunc {
	return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
		logger := getLogger(ctx)
		if logger == nil {
			logger = log.WithFields(log.Fields{"cid": uuid.New()})
		}
		ctx = setLogger(ctx, logger)
		return next(ctx, name, state)
	}
}

func logComponentCheck(next health.InterceptorFunc) health.InterceptorFunc {
	return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
		logger := getLogger(ctx)
		logger.Infof("starting component check")
		res := next(ctx, name, state)
		logger.Infof("component check finished")
		return res
	}
}

func logStatusChange(next health.InterceptorFunc) health.InterceptorFunc {
	return func(ctx context.Context, name string, state health.CheckState) health.CheckState {
		oldStatus := state.Status
		res := next(ctx, name, state)
		if oldStatus != res.Status {
			getLogger(ctx).Warnf("status changed from %s to %s", oldStatus, res.Status)
		}
		return res
	}
}

func logCheck(next health.MiddlewareFunc) health.MiddlewareFunc {
	return func(ctx context.Context) health.CheckerResult {
		logger := getLogger(ctx)
		logger.Infof("starting system check")
		res := next(ctx)
		logger.Infof("system check finished")
		return res
	}
}

func createSystemLogger(next health.MiddlewareFunc) health.MiddlewareFunc {
	return func(ctx context.Context) health.CheckerResult {
		logger := getLogger(ctx)
		if logger == nil {
			logger = log.WithFields(log.Fields{"cid": uuid.New()})
		}
		ctx = setLogger(ctx, logger)
		return next(ctx)
	}
}

func logSystemStatusChange(ctx context.Context, state health.CheckerState) {
	getLogger(ctx).Warnf("status changed to %s", state.Status)
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
