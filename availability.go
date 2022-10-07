package health

// AvailabilityStatus expresses the availability of either
// a component or the whole system.
type AvailabilityStatus string

const (
	// StatusUnknown holds the information that the availability
	// status is not known, because not all checks were executed yet.
	StatusUnknown AvailabilityStatus = "unknown"
	// StatusUp holds the information that the system or a component
	// is up and running.
	StatusUp AvailabilityStatus = "up"
	// StatusDown holds the information that the system or a component
	// down and not available.
	StatusDown AvailabilityStatus = "down"
)

func (s AvailabilityStatus) criticality() int {
	switch s {
	case StatusDown:
		return 2
	case StatusUnknown:
		return 1
	default:
		return 0
	}
}

func (s AvailabilityStatus) toPrometheusInt() int {
	switch s {
	case StatusDown:
		return 0
	case StatusUp:
		return 1
	default:
		return 0
	}
}
