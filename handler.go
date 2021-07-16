package health

import (
	"encoding/json"
	"net/http"
)

type (
	Handler struct {
		ckr            Checker
		statusCodeUp   int
		statusCodeDown int
	}
)

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Do the check
	res := h.ckr.Check(r.Context())

	jsonResp, err := json.Marshal(res)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Write HTTP response
	disableResponseCache(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(mapHTTPStatus(res.Status, h.statusCodeUp, h.statusCodeDown))
	w.Write(jsonResp)
}

func disableResponseCache(w http.ResponseWriter) {
	// The response must be explicitly defined as "not cacheable"
	// to avoid returning an incorrect AvailabilityStatus as a result of caching network equipment.
	// refer to https://www.ibm.com/garage/method/practices/manage/health-check-apis/
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "-1")
}

// NewHandler creates a new health check http.Handler. If periodic checks have
// been configured (see WithPeriodicCheck), they will be started as well
// (if not explicitly turned off using WithManualStart).
func NewHandler(checker Checker) Handler {
	return Handler{ckr: checker, statusCodeUp: http.StatusOK, statusCodeDown: http.StatusServiceUnavailable}
}

// NewHandlerWithStatusCodes creates a new health check http.Handler. If periodic checks have
// been configured (see WithPeriodicCheck), they will be started as well
// (if not explicitly turned off using WithManualStart).
func NewHandlerWithStatusCodes(checker Checker, statusCodeUp int, statusCodeDown int) Handler {
	return Handler{ckr: checker, statusCodeUp: statusCodeUp, statusCodeDown: statusCodeDown}
}

func mapHTTPStatus(status AvailabilityStatus, statusCodeUp int, statusCodeDown int) int {
	if status == StatusDown || status == StatusUnknown {
		return statusCodeDown
	}
	return statusCodeUp
}
