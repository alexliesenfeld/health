<div align="center">
    <h1>Health</h1>
</div>

<p align="center">A simple and flexible health check library for Go.</p>
<div align="center">

[![Build](https://github.com/alexliesenfeld/health/actions/workflows/build.yml/badge.svg)](https://github.com/alexliesenfeld/health/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/alexliesenfeld/health/branch/main/graph/badge.svg?token=V2mVh8RvYE)](https://codecov.io/gh/alexliesenfeld/health)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexliesenfeld/health)](https://goreportcard.com/report/github.com/alexliesenfeld/health)
[![GolangCI](https://golangci.com/badges/github.com/alexliesenfeld/health.svg)](https://golangci.com/r/github.com/alexliesenfeld/health)
[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth.svg?type=shield)](https://app.fossa.com/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth?ref=badge_shield)

</div>

<p align="center">
    <a href="https://pkg.go.dev/github.com/alexliesenfeld/health">Documentation</a>
    ·
    <a href="https://github.com/alexliesenfeld/health/issues">Report Bug</a>
    ·
    <a href="https://github.com/alexliesenfeld/health/issues">Request Feature</a>
</p>

## Features

This library allows you to build health checks that do not simply return HTTP status code 200 but actually check if all
necessary components are healthy.

This library provides the following features:

- [Request based](https://pkg.go.dev/github.com/alexliesenfeld/health#WithCheck) ("synchronous") and 
  [fixed-schedule](https://pkg.go.dev/github.com/alexliesenfeld/health#WithPeriodicCheck) ("asynchronous") health checks.
- Timeout management.
- Health [status change listeners](https://pkg.go.dev/github.com/alexliesenfeld/health#WithStatusListener).
- [Flexible lifecycle hooks]()
- [Caching](https://pkg.go.dev/github.com/alexliesenfeld/health#WithCacheDuration)
- [Failure tolerance](https://pkg.go.dev/github.com/alexliesenfeld/health#readme-failure-tolerance) based on fail count and/or time thresholds.
- Provides an [http.Handler](https://golang.org/pkg/net/http/#Handler) and 
  [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) that are fully compatible with 
  [net/http](https://golang.org/pkg/net/http/#ServeMux).

[This example](https://github.com/alexliesenfeld/health/blob/main/examples/showcase/main.go)
showcases **all features** of this library.

## Getting Started

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alexliesenfeld/health"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"time"
)

func main() {
	db, _ := sql.Open("sqlite3", "simple.sqlite")
	defer db.Close()

	// Create a new Checker
	checker := health.NewChecker(

		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// A simple check to see if database connection is up.
		health.WithCheck(health.Check{
			Name:    "database",
			Timeout: 2 * time.Second, // A check specific timeout.
			Check:   db.PingContext,
		}),

		// The following check will be executed periodically every 30 seconds 
		// started without an initial delay.
		health.WithPeriodicCheck(30*time.Second, 0, health.Check{
			Name: "search",
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this makes the check fail")
			},
		}),
	)

	// We Create a new http.Handler that provides health check information
	// serialized as a JSON string via HTTP.
	http.Handle("/health", health.NewHandler(checker))
	http.ListenAndServe(":3000", nil)
}
```

Because our search component is down, the request `curl -u username:password http://localhost:3000/health` 
would yield a response with HTTP status code `503 (Service Unavailable)`, and the following JSON response body:

```json
{
  "status": "down",
  "details": {
    "database": {
      "status": "up",
      "timestamp": "2021-07-01T08:05:14.603364Z"
    },
    "search": {
      "status": "down",
      "timestamp": "2021-07-01T08:05:08.522685Z",
      "error": "this makes the check fail"
    }
  }
}
```

## Periodic Checks

With "synchronous" health checks we mean that every HTTP request initiates a health check and waits
until all check functions complete before returning an aggregated health status. This approach is usually OK 
for smaller applications with a low number of quickly checkable dependencies. However, it will not scale well 
enough for more involved applications that either have many dependencies and/or some relatively slow check functions.

Rather than executing health check functions on every HTTP request, periodic (or "asynchronous") 
health checks execute the check function on a fixed schedule. With this approach, the health status is always 
read from a local cache that is regularly updated in the background. This allows responding to HTTP requests 
instantly without waiting for the check function to complete. 

Periodic checks can be configured using the `WithPeriodicCheck` configuration option 
(see [example above](#getting-started)). 

You can mix synchronous and asynchronous checks in your application. 

## Caching

Health check responses are cached to avoid sending too many request to the services that your program checks and to
mitigate "denial of service" attacks. The [TTL](https://en.wikipedia.org/wiki/Time_to_live) is set to 1 second by
default. If you do not want to use caching altogether, you can disable it using the
`health.WithDisabledCache()` configuration option.

## Failure Tolerance

This library lets you configure failure tolerant checks that allow some degree of failure. The check is only considered
failed, when predefined tolerance thresholds are crossed.

### Example

Let's assume that your app provides a REST API but also consumes messages from a Kafka topic. If the connection to Kafka
is down, your app can still serve API requests, but it will not process any messages during this time. If the Kafka
health check is configured without any failure tolerance, your whole application will become unhealthy. 
This is most likely not what you want. However, if Kafka is down for too long, there
may indeed be a problem that requires attention. In this case, you still may want to flag your app unhealthy by
returning a failing health check, so that it can be automatically restarted by your infrastructure.

Failure tolerant health checks let you configure this kind of behaviour.

```go
health.WithCheck(health.Check{
    Name:    "unreliable-service",
    // Check is allowed to fail up to 4 times until considered unavailable
    MaxContiguousFails: 4,
    // Check is allowed to be in an erroneous state for up to 1 minute until considered unavailable.
    MaxTimeInError:      1 * time.Minute,
    Check: myCheckFunc,
}),
```

## Hooks
### Listening to Status Changes

It can be useful to react to health status changes. For example, you might want to log status changes, so you can 
easier correlate logs during root cause analysis or perform actions to mitigate the impact of an unhealthy component.

This library allows you to configure listener functions that will be called when either the overall/aggregated health
status changes, or that of a specific component.

#### Example
```go
health.WithPeriodicCheck(5*time.Second, health.Check{
    Name:   "search",
    Check:  myCheckFunc,
    StatusListener: func(ctx context.Context, name string, state CheckState) ) {
        log.Printf("status of component '%s' changed to %s", name, state.Status)
    },
}),

health.WithStatusListener(func(ctx context.Context, state CheckerState)) {
    log.Printf("overall system health status changed to %s", state.Status)
}),
```

### Middleware and Interceptors

It can be useful to hook into the checking lifecycle to do some processing before and after a health check. 
For example, you might want to add some tracing information to the [Context](https://pkg.go.dev/context#Context) before
the check function executes, do some logging or modify the check result before sending the HTTP response 
(e.g., removing details on failed authentication). 

This library provides two mechanisms that allow you to hook into processing:

* [Interceptors](https://pkg.go.dev/github.com/alexliesenfeld/health#InterceptorFunc) make it possible to 
  intercept all calls to a check function. This is useful if you have cross-functional code that needs to be reusable 
  and should have access to check state information.   
* [Middleware](https://pkg.go.dev/github.com/alexliesenfeld/health#MiddlewareFunc) gives you the possibility 
  to intercept all calls of [Checker.Check](https://pkg.go.dev/github.com/alexliesenfeld/health#Checker), which 
  corresponds to every incoming HTTP request. In contrary to the usually used 
  [middleware pattern](https://drstearns.github.io/tutorials/gomiddleware/), this middleware allows you to access 
  check related information and post-process a check result before sending it in an HTTP response.

## License

`health` is free software: you can redistribute it and/or modify it under the terms of the MIT Public License.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the MIT Public License for more details.

[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth.svg?type=large)](https://app.fossa.com/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth?ref=badge_large)
