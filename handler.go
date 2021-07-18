package health

import (
	"encoding/json"
	"net/http"
)

type HandlerConfig struct {
	StatusCodeUp            int
	StatusCodeDown          int
	DisableCheckerAutostart bool
}

// NewHandler creates a new health check http.Handler.
// The Checker will be started automatically (see Checker.Start).
func NewHandler(checker Checker) http.Handler {
	return NewHandlerFunc(checker)
}

// NewHandlerWithConfig creates a new health check http.Handler.
// If HandlerConfig.DisableCheckerAutostart is not true,
// the Checker will be started automatically (see Checker.Start).
func NewHandlerWithConfig(checker Checker, cfg HandlerConfig) http.Handler {
	return NewHandlerFuncWithConfig(checker, cfg)
}

// NewHandlerFunc creates a new health check http.Handler.
// The Checker will be started automatically (see Checker.Start).
func NewHandlerFunc(checker Checker) http.HandlerFunc {
	return NewHandlerFuncWithConfig(checker, HandlerConfig{http.StatusOK, http.StatusServiceUnavailable, false})
}

// NewHandlerFuncWithConfig creates a new health check http.Handler.
// If HandlerConfig.DisableCheckerAutostart is not true,
// the Checker will be started automatically (see Checker.Start).
func NewHandlerFuncWithConfig(checker Checker, cfg HandlerConfig) http.HandlerFunc {
	if !cfg.DisableCheckerAutostart {
		checker.Start()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Do the check
		res := checker.Check(r.Context())

		jsonResp, err := json.Marshal(res)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Write HTTP response
		disableResponseCache(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(mapHTTPStatus(res.Status, cfg.StatusCodeUp, cfg.StatusCodeDown))
		w.Write(jsonResp)
	}
}

func disableResponseCache(w http.ResponseWriter) {
	// The response must be explicitly defined as "not cacheable"
	// to avoid returning an incorrect AvailabilityStatus as a result of caching network equipment.
	// refer to https://www.ibm.com/garage/method/practices/manage/health-check-apis/
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "-1")
}

func mapHTTPStatus(status AvailabilityStatus, statusCodeUp int, statusCodeDown int) int {
	if status == StatusDown || status == StatusUnknown {
		return statusCodeDown
	}
	return statusCodeUp
}
