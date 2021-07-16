package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type (
	healthCheckConfig struct {
		detailsDisabled      bool
		timeout              time.Duration
		statusCodeUp         int
		statusCodeDown       int
		checks               map[string]*Check
		maxErrMsgLen         uint
		cacheTTL             time.Duration
		withManualStart      bool
		statusChangeListener SystemStatusListener
	}

	defaultChecker struct {
		mtx      sync.Mutex
		cfg      healthCheckConfig
		state    map[string]CheckState
		status   AvailabilityStatus
		endChans []chan *sync.WaitGroup
	}

	checkResult struct {
		checkName string
		newState  CheckState
	}

	// Checker is the main checker interface.
	Checker interface {
		// Start starts all periodic checks and prepares the
		// checker for accepting check requests.
		Start()
		// Stop stops all periodic checks.
		Stop()
		// Check performs a health check. The context may contain
		// deadlines to which will be adhered to and will be
		// passed to downstream calls.
		Check(ctx context.Context) SystemStatus
	}

	// SystemStatus holds the aggregated system health information.
	SystemStatus struct {
		// Status is the aggregated availability status of the system.
		Status AvailabilityStatus `json:"status"`
		// Details contains health information about all checked components.
		Details *map[string]CheckStatus `json:"details,omitempty"`
	}

	// CheckStatus holds the a components health information.
	CheckStatus struct {
		// Status is the availability status of a component.
		Status AvailabilityStatus `json:"status"`
		// Timestamp holds the time when the check happened.
		Timestamp time.Time `json:"timestamp,omitempty"`
		// Error contains the error message, if a check was not successful.
		Error *string `json:"error,omitempty"`
	}

	// CheckState contains all state attributes of a components check.
	CheckState struct {
		// LastCheckedAt holds the time of when the check was last executed.
		LastCheckedAt time.Time
		// LastCheckedAt holds the last time of when the check was "up".
		LastSuccessAt time.Time
		// LastResult holds the error of the last check (is nil if successful).
		LastResult error
		// ConsecutiveFails holds the number of how often the check failed in a row.
		ConsecutiveFails uint
		// The current availability status of the check.
		Status    AvailabilityStatus
		startedAt time.Time
	}

	// SystemStatusListener is a callback function that will be called
	// when the system availability status changes (e.g. from "up" to "down").
	SystemStatusListener func(status AvailabilityStatus, state map[string]CheckState)

	// CheckStatusListener is a callback function that will be called
	// when a components availability status changes (e.g. from "up" to "down").
	CheckStatusListener func(name string, state CheckState)

	// AvailabilityStatus expresses the availability of either
	// a component or the whole system.
	AvailabilityStatus string
)

const (
	// StatusUnknown holds the information that the availability
	// status is not known yet, because no check was yet.
	StatusUnknown AvailabilityStatus = "unknown"
	// StatusUp holds the information that the system or component
	// is available.
	StatusUp AvailabilityStatus = "up"
	// StatusDown holds the information that the system or component
	// is not available.
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

func newDefaultChecker(cfg healthCheckConfig) *defaultChecker {
	state := map[string]CheckState{}
	for _, check := range cfg.checks {
		state[check.Name] = CheckState{
			startedAt: time.Now(),
			Status:    StatusUnknown,
		}
	}
	return &defaultChecker{sync.Mutex{}, cfg, state, StatusUnknown, []chan *sync.WaitGroup{}}
}

func (ck *defaultChecker) Start() {
	ck.mtx.Lock()
	defer ck.mtx.Unlock()

	for _, check := range ck.cfg.checks {
		if isPeriodicCheck(check) {
			var wg *sync.WaitGroup
			endChan := make(chan *sync.WaitGroup, 1)
			ck.endChans = append(ck.endChans, endChan)
			go func(check Check, currentState CheckState) {
			loop:
				for {
					currentState = doCheck(context.Background(), check, currentState).newState
					ck.mtx.Lock()
					ck.updateState(checkResult{check.Name, currentState})
					ck.mtx.Unlock()
					select {
					case <-time.After(check.updateInterval):
					case wg = <-endChan:
						break loop
					}
				}
				close(endChan)
				wg.Done()
			}(*check, ck.state[check.Name])
		}
	}
}

func (ck *defaultChecker) Stop() {
	ck.mtx.Lock()

	var wg sync.WaitGroup
	for _, endChan := range ck.endChans {
		wg.Add(1)
		endChan <- &wg
	}

	ck.endChans = []chan *sync.WaitGroup{}
	ck.mtx.Unlock()
	wg.Wait()
}

func (ck *defaultChecker) Check(ctx context.Context) SystemStatus {
	ck.mtx.Lock()
	defer ck.mtx.Unlock()

	ctx, cancel := context.WithTimeout(ctx, ck.cfg.timeout)
	defer cancel()

	var (
		numChecks          = len(ck.cfg.checks)
		resChan            = make(chan checkResult, numChecks)
		cacheTTL           = ck.cfg.cacheTTL
		numInitiatedChecks = 0
	)

	for _, c := range ck.cfg.checks {
		state := ck.state[c.Name]
		if !isPeriodicCheck(c) && isCacheExpired(cacheTTL, &state) {
			numInitiatedChecks++
			go func(ctx context.Context, check Check, state CheckState) {
				resChan <- doCheck(ctx, check, state)
			}(ctx, *c, state)
		}
	}

	var results []checkResult
	for len(results) < numInitiatedChecks {
		results = append(results, <-resChan)
	}

	ck.updateState(results...)

	return newSystemStatus(ck.status, ck.stateToCheckResult(), !ck.cfg.detailsDisabled)
}

func (ck *defaultChecker) updateState(updates ...checkResult) {
	for _, update := range updates {
		ck.updateCheckState(update)
	}

	oldStatus := ck.status
	ck.status = aggregateStatus(ck.state)

	if oldStatus != ck.status && ck.cfg.statusChangeListener != nil {
		ck.cfg.statusChangeListener(ck.status, ck.state)
	}
}

func (ck *defaultChecker) updateCheckState(res checkResult) {
	var (
		name      = res.checkName
		newState  = res.newState
		oldStatus = ck.state[name].Status
		listener  = ck.cfg.checks[name].StatusListener
	)

	ck.state[name] = newState
	if listener != nil && oldStatus != newState.Status {
		listener(name, newState)
	}
}

func (ck *defaultChecker) stateToCheckResult() map[string]CheckStatus {
	results := map[string]CheckStatus{}
	for _, c := range ck.cfg.checks {
		state := ck.state[c.Name]
		results[c.Name] = newCheckStatus(&state, ck.cfg.maxErrMsgLen)
	}
	return results
}

func newSystemStatus(status AvailabilityStatus, results map[string]CheckStatus, withDetails bool) SystemStatus {
	aggStatus := SystemStatus{Status: status}
	if withDetails {
		aggStatus.Details = &results
	}
	return aggStatus
}

func isCacheExpired(cacheDuration time.Duration, state *CheckState) bool {
	return state.LastCheckedAt.Before(time.Now().Add(-cacheDuration))
}

func isPeriodicCheck(check *Check) bool {
	return check.updateInterval > 0
}

func doCheck(ctx context.Context, check Check, state CheckState) checkResult {
	cancel := func() {}
	if check.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, check.Timeout)
	}
	defer cancel()

	state.LastResult = executeCheckFunc(ctx, &check)
	state.LastCheckedAt = time.Now().UTC()

	if state.LastResult == nil {
		state.ConsecutiveFails = 0
		state.LastSuccessAt = state.LastCheckedAt
	} else {
		state.ConsecutiveFails++
	}

	state.Status = evaluateCheckStatus(&state, check.MaxTimeInError, check.MaxConsecutiveFails)

	return checkResult{check.Name, state}
}

func executeCheckFunc(ctx context.Context, check *Check) error {
	res := make(chan error)
	go func() {
		res <- check.Check(ctx)
	}()

	select {
	case r := <-res:
		return r
	case <-ctx.Done():
		return fmt.Errorf("check timed out")
	}
}

func newCheckStatus(state *CheckState, maxErrMsgLen uint) CheckStatus {
	return CheckStatus{
		Status:    state.Status,
		Error:     toErrorDesc(state.LastResult, maxErrMsgLen),
		Timestamp: state.LastCheckedAt,
	}
}

func toErrorDesc(err error, maxLen uint) *string {
	if err != nil {
		errDesc := err.Error()
		if uint(len(errDesc)) > maxLen {
			errDesc = errDesc[:maxLen]
		}
		return &errDesc
	}
	return nil
}

func evaluateCheckStatus(state *CheckState, maxTimeInError time.Duration, maxFails uint) AvailabilityStatus {
	if state.LastCheckedAt.IsZero() {
		return StatusUnknown
	} else if state.LastResult != nil {
		maxTimeInErrorSinceStartPassed := state.startedAt.Add(maxTimeInError).Before(time.Now())
		maxTimeInErrorSinceLastSuccessPassed := state.LastSuccessAt.Add(maxTimeInError).Before(time.Now())

		timeInErrorThresholdCrossed := maxTimeInErrorSinceStartPassed && maxTimeInErrorSinceLastSuccessPassed
		failCountThresholdCrossed := state.ConsecutiveFails >= maxFails

		if failCountThresholdCrossed && timeInErrorThresholdCrossed {
			return StatusDown
		}
	}
	return StatusUp
}

func aggregateStatus(results map[string]CheckState) AvailabilityStatus {
	status := StatusUp
	for _, result := range results {
		if result.Status.criticality() > status.criticality() {
			status = result.Status
		}
	}
	return status
}
