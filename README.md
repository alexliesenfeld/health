<div align="center">
<h1>Health</h1>
</div>

<p align="center">A simple and flexible health check library for Go.</p>
<div align="center">

[![Build Status](https://dev.azure.com/alexliesenfeld/httpmock/_apis/build/status/alexliesenfeld.httpmock?branchName=master)](https://dev.azure.com/alexliesenfeld/httpmock/_build/latest?definitionId=2&branchName=master)
[![codecov](https://codecov.io/gh/alexliesenfeld/httpmock/branch/master/graph/badge.svg)](https://codecov.io/gh/alexliesenfeld/httpmock)
[![crates.io](https://img.shields.io/crates/d/httpmock.svg)](https://crates.io/crates/httpmock)
[![Mentioned in Awesome](https://camo.githubusercontent.com/e5d3197f63169393ee5695f496402136b412d5e3b1d77dc5aa80805fdd5e7edb/68747470733a2f2f617765736f6d652e72652f6d656e74696f6e65642d62616467652e737667)](https://github.com/rust-unofficial/awesome-rust#testing)
[![Go](https://img.shields.io/badge/rust-1.46%2B-blue.svg?maxAge=3600)](https://github.com/rust-lang/rust/blob/master/RELEASES.md#version-1460-2020-08-27)

</div>

<p align="center">
    <a href="https://docs.rs/httpmock/">Documentation</a>
    ·
    <a href="https://github.com/alexliesenfeld/health/issues">Report Bug</a>
    ·
    <a href="https://github.com/alexliesenfeld/health/issues">Request Feature</a>
</p>

## Features
This library allows you to build health checks that do not simply return 200 HTTP status codes but actually 
check if the systems that your app requires to function are actually available.

This library provides the following features:

- Request and check-based timeout management.
- Caching support to unburden checked systems during load peeks.
- Request based and fixed-schedule health checks.
- Custom HTTP request middleware to preprocess and postprocess requests.
- Authentication middleware allows separating public and private health check information.
- Provides an [http.Handler](https://golang.org/pkg/net/http/#Handler) that can be easily used with any [mux](https://golang.org/pkg/net/http/#ServeMux).
- Failure tolerance based on fail count and/or time thresholds.
- Aggregated system status is returned as a JSON body.

This library can be used to easily integrate with the Kubernetes liveness and readiness checks.

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
	router.Handle("/health", health.New(
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
   "checks":{
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
prevent "denial of service" attacks. If you do not want to use caching, you can disable it using the
`health.WithDisabledCache()` configuration option.

## Security
The data returned by health checks often contains sensitive data (such as service names, error messages, etc.).
You probably do not want to expose this information to everyone. For this reason, unauthenticated requests 
only contain the availability status of the service (HTTP response code 200, when available, or 503 if not).
You can enable this functionality by using the built-in basic auth middleware or provide your own authentication 
middleware:

```go
health.New(
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

## Failure Tolerant Checks
Let’s assume that you use a key/value store for caching, and the connection to it is checked by your application as well. 
Your app is capable of running without the key/value store, but it will result in a slowdown. 
If the key/value store is down, your whole application will appear unavailable. This is most likely not what you want.

However, if the connection cannot be restored for a long time, there may be a serious problem that requires attention.
In this case, you still may want to provide a failing health check, so that your app can be automatically restarted 
by your infrastructure and potentially solve the problem
(such as [Kubernetes health checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)). 

## Metrics
This library does not come with built-in metrics. Its focus is on health checks.
Please use a proper metrics framework instead (e.g. Prometheus).

## License
`health` is free software: you can redistribute it and/or modify it under the terms of the MIT Public License.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied 
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the MIT Public License for more details.