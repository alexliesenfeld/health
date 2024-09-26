## 0.8.1
### Improvements
- The configuration option [`WithChecks`](https://pkg.go.dev/github.com/alexliesenfeld/health@v0.8.1#WithChecks)
  was added. 
- The configuration option [`WithInfoFunc`](https://pkg.go.dev/github.com/alexliesenfeld/health@v0.8.1#WithInfoFunc)
  was added.

## 0.8.0
### Breaking Changes
- [`CheckerResult`](https://github.com/alexliesenfeld/health/blob/8d498ec975b54ec3ef47493bbc22c72884359dc2/check.go#L86C1-L91)s 
`Details` field is now no pointer anymore.
- The configuration option [`WithMaxErrorMessageLength`](https://pkg.go.dev/github.com/alexliesenfeld/health@v0.7.0#WithMaxErrorMessageLength) 
was removed. This used to control the length of the string field [`CheckResult.Error`](https://pkg.go.dev/github.com/alexliesenfeld/health@v0.7.0#CheckResult).
Instead of returning the error as a string, it is now being returned as an `error`.
- All [`time.Time`](https://pkg.go.dev/time#Time) fields in [`health.CheckState`](https://pkg.go.dev/github.com/alexliesenfeld/health@v0.7.0#CheckState)
  are now values rather than pointers. Use the
[`IsZero`](https://pkg.go.dev/time#Time.IsZero)-method to check if a value has been set or not instead.

## 0.7.0
### Breaking Changes
- This version introduces automatic recovery from panics that can be turned off on a per-check basis like shown in the [showcase example](https://github.com/alexliesenfeld/health/blob/1fcc4c7599ea00dbd0c73c97448b2a1c1d0fff7d/examples/showcase/main.go#L92-L95).
- A bug has been fixed that could cause goroutine leaks for timed out check functions.

### Improvements
- The initial check run that is executed on startup is non-blocking anymore.

## 0.6.0 
### Breaking Changes
- A [ResultWriter](https://pkg.go.dev/github.com/alexliesenfeld/health#ResultWriter) must now additionally write the 
  status code into the [http.ResponseWriter](https://pkg.go.dev/net/http#ResponseWriter). This is necessary due to 
  ordering constraints when writing into a [http.ResponseWriter](https://pkg.go.dev/net/http#ResponseWriter) 
  (see https://github.com/alexliesenfeld/health/issues/9).
  
### Improvements
- [Stopping the Checker](https://pkg.go.dev/github.com/alexliesenfeld/health#Checker) does not wait until the 
  [initial delay of periodic checks](https://pkg.go.dev/github.com/alexliesenfeld/health#WithPeriodicCheck)
  has passed anymore. [Checker.Stop](https://pkg.go.dev/github.com/alexliesenfeld/health#Checker) stops
  the [Checker](https://pkg.go.dev/github.com/alexliesenfeld/health#Checker) immediately, but waits until all currently 
  running check functions have completed.
- The [health check http.Handler](https://pkg.go.dev/github.com/alexliesenfeld/health#NewHandler) was patched to not 
  include an empty `checks` map in the JSON response. In case no check functions are defined, the JSON response will 
  therefore not be `{ "status": "up", "checks" : {} }` anymore but only `{ "status": "up" }`. 
- A Kubernetes liveness and readiness checks example was added (see `examples/kubernetes`).

## 0.5.1
- Many documentation improvements

## 0.5.0

- BREAKING CHANGE: Changed function signature of middleware functions.
- Added a new check function interceptor and a [http.Handler](https://pkg.go.dev/net/http#Handler) 
  middleware with basic logging functionality.
- Added a new basic authentication middleware that reduces the exposed health information in case of 
  failed authentication.
- Added a new middleware FullDetailsOnQueryParam was added that hides details by default and only shows 
  them when a configured query parameter name was provided in the HTTP request.
- Added new Checker configuration option WithInterceptors, that will be applied to every check function.
