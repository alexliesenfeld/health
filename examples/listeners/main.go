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

		// A simple exampleSuccessfulCheckFunc to see if database connection is up.
		health.WithCheck(health.Check{
			Name:                 "database",
			Timeout:              2 * time.Second, // A a exampleSuccessfulCheckFunc specific timeout.
			Check:                exampleSuccessfulCheckFunc,
			BeforeCheckListener:  logBeforeComponentCheck,
			AfterCheckListener:   logAfterComponentCheck,
			StatusChangeListener: logComponentStatusChanged,
		}),

		// A simple exampleSuccessfulCheckFunc to see if database connection is up.
		health.WithCheck(health.Check{
			Name:                 "filesystem",
			Timeout:              2 * time.Second, // A a exampleSuccessfulCheckFunc specific timeout.
			Check:                exampleFailingCheckFunc,
			BeforeCheckListener:  logBeforeComponentCheck,
			AfterCheckListener:   logAfterComponentCheck,
			StatusChangeListener: logComponentStatusChanged,
		}),

		// The following exampleSuccessfulCheckFunc will be executed periodically every 30 seconds.
		health.WithPeriodicCheck(15*time.Second, health.Check{
			Name:                 "elasticsearch",
			Check:                exampleSuccessfulCheckFunc,
			BeforeCheckListener:  logBeforeComponentCheck,
			AfterCheckListener:   logAfterComponentCheck,
			StatusChangeListener: logComponentStatusChanged,
		}),

		health.WithBeforeCheckListener(logBeforeCheck),
		health.WithAfterCheckListener(logAfterCheck),
		health.WithStatusListener(logStatusChanged),
	)

	// We Create a new http.Handler that provides health exampleSuccessfulCheckFunc information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))

	http.ListenAndServe(":3000", nil)
}

// **************************************************************
// Event listener functions
// **************************************************************
func logBeforeCheck(ctx context.Context, state map[string]health.CheckState) context.Context {
	logger := log.WithFields(log.Fields{"cid": uuid.New()})
	logger.Info("starting new check")
	return setLogger(ctx, logger)
}

func logAfterCheck(ctx context.Context, state map[string]health.CheckState) context.Context {
	getLogger(ctx).Info("finished check")
	return ctx
}

func logStatusChanged(ctx context.Context, status health.AvailabilityStatus, state map[string]health.CheckState) {
	getLogger(ctx).Infof("system status changed to %s", status)
}

func logBeforeComponentCheck(ctx context.Context, name string, state health.CheckState) context.Context {
	// Please note that a BeforeCheckListener (as defined in WithBeforeCheckListener, see our function
	// "logBeforeCheck" above) and AfterCheckListeners are not executed for periodic checks but only
	// for non-periodic checks. To be able to use this function for both, periodic and non-periodic checks,
	// we create a new logger if we don't find one in the context.
	logger := getLogger(ctx)
	if logger == nil {
		logger = log.WithFields(log.Fields{"cid": uuid.New()})
	}
	logger = logger.WithFields(log.Fields{"component": name})
	logger.Info("starting component check")
	return setLogger(ctx, logger)
}

func logAfterComponentCheck(ctx context.Context, state health.CheckState) context.Context {
	logger := getLogger(ctx)
	if state.LastResult != nil {
		getLogger(ctx).Warnf("ended component check with error: %v", state.LastResult.Error())
	} else {
		logger.Info("ended component check with success")
	}
	return ctx
}

func logComponentStatusChanged(ctx context.Context, state health.CheckState) {
	getLogger(ctx).Infof("component changed status to %s", state.Status)
}

// **************************************************************
// Helper functions
// **************************************************************
func exampleSuccessfulCheckFunc(ctx context.Context) error {
	return nil
}

func exampleFailingCheckFunc(ctx context.Context) error {
	return fmt.Errorf("this is a check error")
}

func getLogger(ctx context.Context) *log.Entry {
	v, ok := ctx.Value("logger").(*log.Entry)
	if ok {
		return v
	}
	return nil
}

func setLogger(ctx context.Context, logger *log.Entry) context.Context {
	return context.WithValue(ctx, "logger", logger)
}
