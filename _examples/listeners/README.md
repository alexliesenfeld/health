This example shows how listeners can be used to hook into checking lifecycle.
```go
package main

import (
	"context"
	"fmt"
	"github.com/alexliesenfeld/health"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {

	// Create a new Checker
	checker := health.NewChecker(

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// A simple check to see if database connection is up.
		health.WithCheck(health.Check{
			Name:                "database",
			Timeout:             2 * time.Second, // A a check specific timeout.
			BeforeCheckListener: beforeCheckListener("database"),
			AfterCheckListener:  afterCheckListener("database"),
			StatusListener:      componentStatusListener("database"),
			Check:               checkFunc("database"),
		}),

		// The following check will be executed periodically every 30 seconds.
		health.WithPeriodicCheck(15*time.Second, health.Check{
			Name:                "search",
			BeforeCheckListener: beforeCheckListener("search"),
			AfterCheckListener:  afterCheckListener("search"),
			StatusListener:      componentStatusListener("search"),
			Check:               checkFunc("search"),
		}),

		health.WithBeforeCheckListener(func(ctx context.Context, state map[string]health.CheckState) context.Context {
			sysRun := rand.Intn(1000000)
			log.Println(fmt.Sprintf("%d: starting a system check", sysRun))
			return context.WithValue(ctx, "sysRun", sysRun)
		}),

		health.WithAfterCheckListener(func(ctx context.Context, state map[string]health.CheckState) context.Context {
			log.Println(fmt.Sprintf("%d: system check ended", ctx.Value("sysRun")))
			return ctx
		}),

		health.WithStatusListener(func(c context.Context, a health.AvailabilityStatus, s map[string]health.CheckState) {
			log.Println(fmt.Sprintf("%d: system status changed to %s", c.Value("sysRun"), a))
		}),
	)

	// We Create a new http.Handler that provides health check information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))

	http.ListenAndServe(":3000", nil)
}

func beforeCheckListener(name string) func(ctx context.Context, state health.CheckState) context.Context {
	return func(ctx context.Context, state health.CheckState) context.Context {
		run := rand.Intn(1000000)
		log.Println(fmt.Sprintf("%d | %d | %s: starting a new check run", ctx.Value("sysRun"), run, name))
		return context.WithValue(ctx, name, run)
	}
}

func afterCheckListener(name string) func(ctx context.Context, state health.CheckState) context.Context {
	return func(ctx context.Context, state health.CheckState) context.Context {
		log.Println(fmt.Sprintf("%d | %d | %s: ended check run ", ctx.Value("sysRun"), ctx.Value(name), name))
		return ctx
	}
}

func checkFunc(name string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		log.Println(fmt.Sprintf("%d | %d | %s: starting check func", ctx.Value("sysRun"), ctx.Value(name), name))
		time.Sleep(250 * time.Millisecond)
		log.Println(fmt.Sprintf("%d | %d | %s: ended check func", ctx.Value("sysRun"), ctx.Value(name), name))
		return nil
	}
}

func componentStatusListener(name string) func(ctx context.Context, state health.CheckState) {
	return func(ctx context.Context, state health.CheckState) {
		log.Println(fmt.Sprintf("%s: component check status changed to %s in run %d", name, state.Status, ctx.Value(name)))
	}
}
```