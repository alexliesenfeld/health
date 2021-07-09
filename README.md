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
This library allows you to build health checks that do not simply return 200 HTTP status codes but actually 
check if the systems that your app requires to work are actually available.

This library provides the following features:

- Request based and fixed-schedule health checks.
- Request and check-based timeout management.
- Caching support to unburden checked systems during load peeks.
- Custom HTTP request middleware to preprocess and postprocess requests.
- Authentication middleware allows separating public and private health check information.
- Provides an [http.Handler](https://golang.org/pkg/net/http/#Handler) that can be easily used with any [mux](https://golang.org/pkg/net/http/#ServeMux).
- Failure tolerance based on fail count and/or time thresholds.

This library can be used to integrate with the Kubernetes liveness and readiness checks.

## Example
```go
package main

import (
	"context"
	"database/sql"
	"github.com/alexliesenfeld/health"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"time"
)

func main() {
	db, _ := sql.Open("sqlite3", "simple.sqlite")
	defer db.Close()

	router := http.NewServeMux()
	router.Handle("/health", health.NewHandler(
		health.WithTimeout(10*time.Second),
		health.WithBasicAuth("username", "password", true),
		health.WithCheck(health.Check{
			Name:  "database",
			Check: db.PingContext,
		}),
		health.WithPeriodicCheck(30*time.Second, health.Check{
			Name: "search",
			Timeout: 5*time.Second,
			Check: func(ctx context.Context) error {
				_, err := http.Get("https://www.google.com")
				return err
			},
		}),
	))

	http.ListenAndServe(":3000", router)
}
```

The request `curl -u username:password http://localhost:3000/health` would then yield the following result:

```json
{
   "status":"DOWN",
   "timestamp":"2021-07-01T08:05:08.522685Z",
   "details":{
      "database":{
         "status":"DOWN",
         "timestamp":"2021-07-01T08:05:14.603364Z",
         "error" : "check timed out"
      },
      "search":{
         "status":"UP",
         "timestamp":"2021-07-01T08:05:08.522685Z"
      }
   }
}
```

## Caching
Health responses are cached to avoid burdening the services that your program checks and to
mitigate "denial of service" attacks. Caching can be configured globally and/or be fine-tuned per check. 
If you do not want to use caching altogether, you can disable it using the `health.WithDisabledCache()` 
configuration option.

## Security
The data that is returned as part of health check results usually contains sensitive information 
(such as service names, error messages, etc.). You probably do not want to expose this information to everyone. 
For this reason, this library provides support for authentication middleware that allows you to hide health details 
or entirely block requests based on authentication success.

Example: Based on the example below, the authentication middleware will respond with a JSON response body that only 
contains the health status, and the corresponding HTTP status code (in this case HTTP status code 503 and JSON response 
body `{ "status":"DOWN" }`).

```go
health.NewHandler(
	health.WithBasicAuth("username", "password", true), 
	health.WithCustomAuth(true, func(r *http.Request) error {
		return fmt.Errorf("this simulates authentication failure")
	}), 
	health.WithCheck(health.Check{
		Name:    "database",
		Check: db.PingContext,
	}), 
)
```

Details, such as error messages, services names, etc. are not exposed to the caller. 
This allows you to open health endpoints to the public but only provide details to authenticated sources.

## Periodic Checks
Rather than executing a health check function on every request that is received over the health endpoint,
periodic checks execute the check function on a fixed schedule. This allows to respond to HTTP requests
instantly without waiting for the check function to complete. This is especially useful if you
either expect a higher request rate on the health endpoint, or your checks take a relatively long time to complete.

```go
health.NewHandler(
	health.WithPeriodicCheck(15*time.Second, health.Check{
		Name:    "slow-check",
		Check:   myLongRunningCheckFunc, // your custom long running check function
	}),
)
```

## Failure Tolerant Checks
This library lets you configure failure tolerant checks that allow some degree of failure. The check is only 
considered failed, when tolerance thresholds have been crossed.

### Example: 
Let's assume that your app provides a REST API but also consumes messages from a Kafka topic. If the connection to Kafka
is down, your app can still serve API requests, but it will not process any messages during this time.
If the Kafka health check is configured without any failure tolerance, and the connection to Kafka is temporarily down, 
your whole application will be considered unavailable. This is most likely not what you want. 
However, if Kafka is down for too long, there may indeed be a problem that requires attention. In this case, 
you still may want to flag your app unhealthy by returning a failing health check, so that your app can be 
automatically restarted by your infrastructure. In this case, you can allow some degree of failure tolerance in your
[check config](https://pkg.go.dev/github.com/alexliesenfeld/health#Check) 
(see attribute `FailureTolerance` and `FailureToleranceThreshold`).

## License
`health` is free software: you can redistribute it and/or modify it under the terms of the MIT Public License.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied 
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the MIT Public License for more details.

[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth.svg?type=large)](https://app.fossa.com/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth?ref=badge_large)
