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

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// A simple successFunc to see if a fake database connection is up.
		health.WithCheck(health.Check{
			Name:                "database",
			Timeout:             2 * time.Second, // A successFunc specific timeout.
			BeforeCheckListener: beforeComponentCheck,
			AfterCheckListener:  afterComponentCheck,
			StatusListener:      onComponentStatusChanged,
			Check: func(ctx context.Context) error {
				return nil // no error
			},
		}),

		// A simple successFunc to see if a fake file system up.
		health.WithCheck(health.Check{
			Name:                "filesystem",
			Timeout:             2 * time.Second, // A successFunc specific timeout.
			BeforeCheckListener: beforeComponentCheck,
			AfterCheckListener:  afterComponentCheck,
			StatusListener:      onComponentStatusChanged,
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this is a check error") // example error
			},
		}),

		// The following check will be executed periodically every 30 seconds.
		health.WithPeriodicCheck(15*time.Second, health.Check{
			Name:                "search-engine",
			BeforeCheckListener: beforeComponentCheck,
			AfterCheckListener:  afterComponentCheck,
			StatusListener:      onComponentStatusChanged,
			Check: func(ctx context.Context) error {
				return nil // no error
			},
		}),

		// This listener will be called whenever system health status changes (e.g., from "up" to "down").
		health.WithStatusListener(onSystemStatusChanged),

		// These two are only triggered when Checker.Check is executed (usually done at the start of the checker and
		// afterwords only by the Handler once for every HTTP request). The two are not executed for periodic checks!
		health.WithBeforeCheckListener(beforeRequest),
		health.WithAfterCheckListener(afterRequest),
	)

	// We Create a new http.Handler that provides health successFunc information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))
	http.ListenAndServe(":3000", nil)
}

// **************************************************************
// Event listener functions
// **************************************************************
func beforeComponentCheck(ctx context.Context, name string, state health.CheckState) context.Context {
	logger := getLogger(ctx).WithFields(log.Fields{"component": name})
	logger.Info("starting component check")
	return setLogger(ctx, logger)
}

func onComponentStatusChanged(ctx context.Context, state health.CheckState) context.Context {
	getLogger(ctx).Infof("component changed status to %s", state.Status)
	return ctx
}

func afterComponentCheck(ctx context.Context, state health.CheckState) {
	if state.Result != nil {
		getLogger(ctx).Warnf("ended component check with error: %v", state.Result)
	} else {
		getLogger(ctx).Info("ended component check with success")
	}
}

func beforeRequest(ctx context.Context, state health.CheckerState) context.Context {
	logger := getLogger(ctx)
	logger.Info("starting system health status check")
	return setLogger(ctx, logger)
}

func afterRequest(ctx context.Context, state health.CheckerState) {
	getLogger(ctx).Info("finished system health status check")
}

func onSystemStatusChanged(ctx context.Context, state health.CheckerState) context.Context {
	getLogger(ctx).Infof("system status changed to %s", state.Status)
	return ctx
}

// **************************************************************
// Helper functions
// **************************************************************
func setLogger(ctx context.Context, logger *log.Entry) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

func getLogger(ctx context.Context) *log.Entry {
	v, ok := ctx.Value("logger").(*log.Entry)
	if ok {
		return v
	}
	return log.WithFields(log.Fields{"cid": uuid.New()})
}
