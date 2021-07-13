package health

import (
	"context"
	"encoding/json"
	"net/http"
)

type (
	healthCheckHandler struct {
		http.Handler
		ckr checker
		cfg healthCheckConfig
	}

	checker interface {
		StartPeriodicChecks()
		StopPeriodicChecks()
		Check(ctx context.Context) aggregatedCheckStatus
	}
)

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
// Attention: This function does not block until all checks have been stopped!
func StopPeriodicChecks(healthHandler http.Handler) {
	ck := healthHandler.(*healthCheckHandler)
	ck.ckr.StopPeriodicChecks()
}

func (h *healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set request timeout
	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.timeout)
	defer cancel()
	r = r.WithContext(ctx)

	// Do the check
	res := h.ckr.Check(r.Context())
	jsonResp, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write HTTP response
	disableResponseCache(w)
	w.WriteHeader(mapHTTPStatus(&h.cfg, res.Status))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(jsonResp)
}

func disableResponseCache(w http.ResponseWriter) {
	// The response must be explicitly defined as "noncacheable"
	// to avoid returning an incorrect status as a result of caching network equipment.
	// refer to https://www.ibm.com/garage/method/practices/manage/health-check-apis/
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "-1")
}

func newHandler(cfg healthCheckConfig, ckr checker) http.Handler {
	return &healthCheckHandler{ckr: ckr, cfg: cfg}
}

func mapHTTPStatus(cfg *healthCheckConfig, status Status) int {
	if status == StatusDown || status == StatusUnknown {
		return cfg.statusCodeDown
	}
	return cfg.statusCodeUp
}
