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
This library allows you to build health checks that do not simply return HTTP status code 200 but actually 
check if all necessary components are healthy.

This library provides the following features:

- Request based and fixed-schedule health checks.
- Global and check-based timeout management.
- Custom HTTP request middleware for pre- and postprocessing HTTP requests and/or responses.
- Failure tolerance based on fail count and/or time thresholds.
- Provides an [http.Handler](https://golang.org/pkg/net/http/#Handler) that can be easily used with a [mux](https://golang.org/pkg/net/http/#ServeMux).
- Authentication middleware that allows to hide sensitive information from the public.
- Caching.

## Example
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

	router := http.NewServeMux()
	
	// Create a new http.Handler that provides health check information.
	router.Handle("/health", health.NewHandler(
		
		// Configure a global timeout that will be applied to all checks.
		health.WithTimeout(10*time.Second),

		// A simple check to see if database connection is up.
		health.WithCheck(health.Check{                          
			Name:  "database",
			Timeout: 2*time.Second, // A check specific timeout.
			Check: db.PingContext,
		}),
		
		// The following check will be executed periodically every 30 seconds.
		health.WithPeriodicCheck(30*time.Second, health.Check{  
			Name: "search",
			Check: func(ctx context.Context) error {
				return fmt.Errorf("this makes the check fail")
			},
		}),
	))

	http.ListenAndServe(":3000", router)
}
```

If our database is down, the request `curl -u username:password http://localhost:3000/health` would 
yield a response with HTTP status code `503 (Service Unavailable)`, and the following JSON response body:

```json
{
   "status": "DOWN",
   "timestamp": "2021-07-01T08:05:08.522685Z",
   "details":{
      "database": {
         "status": "UP",
         "timestamp": "2021-07-01T08:05:14.603364Z"
      },
      "search": {
         "status": "DOWN",
         "timestamp": "2021-07-01T08:05:08.522685Z",
          "error": "this makes the check fail"
      }
   }
}
```

## Periodic Checks
Rather than executing a health check function on every request that is received over the health endpoint,
periodic checks execute the check function on a fixed schedule. This allows to respond to HTTP requests
instantly without waiting for the check function to complete. This is especially useful if you
either expect a higher request rate on the health endpoint, or your checks take a relatively long time to complete.

```go
health.NewHandler(
	health.WithPeriodicCheck(15*time.Second, health.Check{
		Name:    "slow-check",
		Check:   someLongRunningCheckFunc,
	}),
)
```

## Caching
Health check responses are cached to avoid burdening downstream services that your program checks with too many 
requests and to mitigate "denial of service" attacks. Caching can be configured globally and/or be fine-tuned per 
check. The [TTL](https://en.wikipedia.org/wiki/Time_to_live) is set to 1 second by default. 
If you do not want to use caching altogether, you can disable it using the `health.WithDisabledCache()` 
configuration option.

## Security
The data that is returned as part of health check results usually contains sensitive information
(such as service names, error messages, etc.). You probably do not want to expose this information to everyone.
For this reason, this library provides support for authentication middleware that allows you to hide health details
or entirely block requests based on authentication success.

### Example
In the example below, we configure a [basic auth](https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication)
middleware that expects a username and password in each HTTP request. If authentication fails, it will respond
with HTTP status code `503 (Service Unavailable)` and a JSON body that only contains the aggregated health status
(`{ "status":"DOWN" }`).

```go
health.NewHandler(
	health.WithBasicAuth("username", "password", true),
	health.WithCheck(health.Check{
		Name:    "database",
		Check: db.PingContext,
	}), 
)
```

Details, such as error messages, services names, etc. are not exposed to the caller.
This allows you to open health endpoints to the public but only provide details to authenticated sources.

## Failure Tolerant Checks
This library lets you configure failure tolerant checks that allow some degree of failure. The check is only 
considered failed, when tolerance thresholds have been crossed.

### Example
Let's assume that your app provides a REST API but also consumes messages from a Kafka topic. If the connection to Kafka
is down, your app can still serve API requests, but it will not process any messages during this time.
If the Kafka health check is configured without any failure tolerance, and the connection to Kafka is temporarily down, 
your whole application will become unhealthy. This is most likely not what you want. However, if Kafka is down for 
too long, there may indeed be a problem that requires attention. In this case, you still may want to flag your 
app unhealthy by returning a failing health check, so that it can be automatically restarted by your infrastructure. 

To allow some failure, please have a look at the `FailureTolerance` and `FailureToleranceThreshold` 
attributes in your [check configuration](https://pkg.go.dev/github.com/alexliesenfeld/health#Check).

## License
`health` is free software: you can redistribute it and/or modify it under the terms of the MIT Public License.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied 
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the MIT Public License for more details.

[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth.svg?type=large)](https://app.fossa.com/projects/custom%2B26405%2Fgithub.com%2Falexliesenfeld%2Fhealth?ref=badge_large)
