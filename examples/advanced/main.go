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
	)

	// We Create a new http.Handler that provides health successFunc information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))
	http.ListenAndServe(":3000", nil)
}

func createLogger(ctx context.Context, name string, state health.CheckState, next health.InterceptorFunc) health.CheckState {
	logger := newGetLogger(ctx)
	if logger == nil {
		logger = log.WithFields(log.Fields{"cid": uuid.New()})
	}
	ctx = newSetLogger(ctx, logger)
	return next(ctx, name, state)
}

func logComponentCheck(ctx context.Context, name string, state health.CheckState, next health.InterceptorFunc) health.CheckState {
	logger := newGetLogger(ctx)
	logger.Infof("starting")
	res := next(ctx, name, state)
	logger.Infof("stopping")
	return res
}

func logStatusChange(ctx context.Context, name string, state health.CheckState, next health.InterceptorFunc) health.CheckState {
	oldStatus := state.Status
	res := next(ctx, name, state)
	if oldStatus != res.Status {
		newGetLogger(ctx).Warnf("status changed from %s to %s", oldStatus, res.Status)
	}
	return res
}

func newSetLogger(ctx context.Context, logger *log.Entry) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

func newGetLogger(ctx context.Context) *log.Entry {
	logger, ok := ctx.Value("logger").(*log.Entry)
	if ok {
		return logger
	}
	return nil
}
