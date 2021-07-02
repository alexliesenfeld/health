package health

import (
	"encoding/json"
	"net/http"
)

type healthCheckHandler struct {
	http.Handler
	ckr checker
}

func (h *healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := h.ckr.Check(r.Context())
	jsonResp, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(mapHTTPStatus(res.Status))
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.Write(jsonResp)
}

func newHandler(middlewares []Middleware, ckr checker) http.Handler {
	var handler http.Handler = &healthCheckHandler{ckr: ckr}
	for _, mw := range middlewares {
		handler = mw(handler)
	}
	return handler
}

func mapHTTPStatus(status availabilityStatus) int {
	if status == statusDown || status == statusUnknown {
		return http.StatusServiceUnavailable
	}
	return http.StatusOK
}

// StartPeriodicChecks allows to start periodic checks manually if the health check was configured using
// WithManualPeriodicCheckStart or when checks have been stopped earlier using health.StopPeriodicChecks.
// This function has no effect otherwise.
func StartPeriodicChecks(healthHandler http.Handler) {
	ck := healthHandler.(*healthCheckHandler)
	ck.ckr.StartPeriodicChecks()
}

// StopPeriodicChecks stops all periodic checks. This function has only effect after automatic startup
// (i.e. when the handler was not configured using WithManualPeriodicCheckStart) or when peridic checks have been
// started before manually using StartPeriodicChecks. This function will have no effect otherwise.
// It is usually not necessary to call this function manually.
func StopPeriodicChecks(healthHandler http.Handler) {
	ck := healthHandler.(*healthCheckHandler)
	ck.ckr.StopPeriodicChecks()
}
