package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type (
	healthCheckConfig struct {
		detailsDisabled           bool
		timeout                   time.Duration
		statusCodeUp              int
		statusCodeDown            int
		checks                    map[string]*Check
		maxErrMsgLen              uint
		cacheTTL                  time.Duration
		beforeSystemCheckListener func(context.Context, CheckerState) context.Context
		statusChangeListener      func(context.Context, CheckerState) context.Context
		afterSystemCheckListener  func(context.Context, CheckerState)
	}

	defaultChecker struct {
		started  bool
		mtx      sync.Mutex
		cfg      healthCheckConfig
		state    CheckerState
		endChans []chan *sync.WaitGroup
	}

	checkResult struct {
		checkName string
		newState  CheckState
	}

	// Checker is the main checker interface. It provides all health checking logic.
	Checker interface {
		// Start will start all necessary background workers and prepare
		// the checker for further usage.
		Start()
		// Stop stops will stop the checker.
		Stop()
		// Check performs a health check. It expects a context, that
		// may contain deadlines to which will be adhered to. The context
		// will be passed to downstream calls.
		Check(ctx context.Context) SystemStatus
		// GetRunningPeriodicCheckCount returns the number of currently
		// running periodic checks.
		GetRunningPeriodicCheckCount() int
	}

	// CheckerState represents the current state of the Checker.
	CheckerState struct {
		// Status is the aggregated system health status.
		Status AvailabilityStatus
		// CheckState contains the state of all checks.
		CheckState map[string]CheckState
	}

	// SystemStatus holds the aggregated system availability status and
	// detailed information about the individual checks.
	SystemStatus struct {
		// Status is the aggregated system availability status.
		Status AvailabilityStatus `json:"status"`
		// Details contains health information for all checked components.
		Details *map[string]CheckStatus `json:"details,omitempty"`
	}

	// CheckStatus holds a components health information.
	CheckStatus struct {
		// Status is the availability status of a component.
		Status AvailabilityStatus `json:"status"`
		// Timestamp holds the time when the check was executed.
		Timestamp *time.Time `json:"timestamp,omitempty"`
		// Error contains the check error message, if the check failed.
		Error *string `json:"error,omitempty"`
	}

	// CheckState represents the current state of a component check.
	CheckState struct {
		// LastCheckedAt holds the time of when the check was last executed.
		LastCheckedAt *time.Time
		// LastCheckedAt holds the last time of when the check did not return an error.
		LastSuccessAt *time.Time
		// LastFailureAt holds the last time of when the check did return an error.
		LastFailureAt *time.Time
		// FirstCheckStartedAt holds the time of when the first check was started.
		FirstCheckStartedAt time.Time
		// ContiguousFails holds the number of how often the check failed in a row.
		ContiguousFails uint
		// Result holds the error of the last check (nil if successful).
		Result error
		// The current availability status of the check.
		Status AvailabilityStatus
	}

	Interceptor     func(ctx context.Context, name string, state CheckState, next InterceptorFunc) CheckState
	InterceptorFunc func(ctx context.Context, name string, state CheckState) CheckState

	// AvailabilityStatus expresses the availability of either
	// a component or the whole system.
	AvailabilityStatus string
)

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

func newDefaultChecker(cfg healthCheckConfig) *defaultChecker {
	checkState := map[string]CheckState{}
	for _, check := range cfg.checks {
		checkState[check.Name] = CheckState{Status: StatusUnknown}
	}
	state := CheckerState{Status: StatusUnknown, CheckState: checkState}
	return &defaultChecker{false, sync.Mutex{}, cfg, state, []chan *sync.WaitGroup{}}
}

func (ck *defaultChecker) Start() {
	ck.mtx.Lock()

	if !ck.started {
		ck.started = true
		defer ck.startPeriodicChecks()
		defer ck.Check(context.Background())

		for _, check := range ck.cfg.checks {
			check.interceptorChain = newInterceptorChain(check)
		}
	}

	ck.mtx.Unlock()
}

func (ck *defaultChecker) startPeriodicChecks() {
	ck.mtx.Lock()
	defer ck.mtx.Unlock()

	// Start periodic checks
	for _, check := range ck.cfg.checks {
		if isPeriodicCheck(check) {
			var wg *sync.WaitGroup
			endChan := make(chan *sync.WaitGroup, 1)
			checkState := ck.state.CheckState[check.Name]
			ck.endChans = append(ck.endChans, endChan)
			go func(check Check, state CheckState) {
			loop:
				for {
					withCheckContext(context.Background(), &check, func(ctx context.Context) {
						ctx, state = doCheck(ctx, &check, state)
						ck.mtx.Lock()
						ck.updateState(ctx, checkResult{check.Name, state})
						ck.mtx.Unlock()
					})
					select {
					case <-time.After(check.updateInterval):
					case wg = <-endChan:
						break loop
					}
				}
				close(endChan)
				wg.Done()
			}(*check, checkState)
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
	ck.started = false

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

	if ck.cfg.beforeSystemCheckListener != nil {
		ctx = ck.cfg.beforeSystemCheckListener(ctx, ck.state)
	}

	for _, c := range ck.cfg.checks {
		checkState := ck.state.CheckState[c.Name]
		if !isPeriodicCheck(c) && isCacheExpired(cacheTTL, &checkState) {
			numInitiatedChecks++
			go func(ctx context.Context, check Check, state CheckState) {
				withCheckContext(ctx, &check, func(ctx context.Context) {
					_, state = doCheck(ctx, &check, state)
					resChan <- checkResult{check.Name, state}
				})
			}(ctx, *c, checkState)
		}
	}

	var results []checkResult
	for len(results) < numInitiatedChecks {
		results = append(results, <-resChan)
	}

	ck.updateState(ctx, results...)

	if ck.cfg.afterSystemCheckListener != nil {
		ck.cfg.afterSystemCheckListener(ctx, ck.state)
	}

	return newSystemStatus(ck.state.Status, ck.mapStateToCheckStatus(), !ck.cfg.detailsDisabled)
}

func (ck *defaultChecker) updateState(ctx context.Context, updates ...checkResult) {
	for _, update := range updates {
		ck.state.CheckState[update.checkName] = update.newState
	}

	oldAggregatedStatus := ck.state.Status
	ck.state.Status = aggregateStatus(ck.state.CheckState)

	if oldAggregatedStatus != ck.state.Status && ck.cfg.statusChangeListener != nil {
		ck.cfg.statusChangeListener(ctx, ck.state)
	}
}

func (ck *defaultChecker) mapStateToCheckStatus() map[string]CheckStatus {
	results := map[string]CheckStatus{}
	for _, c := range ck.cfg.checks {
		checkState := ck.state.CheckState[c.Name]
		results[c.Name] = newCheckStatus(&checkState, ck.cfg.maxErrMsgLen)
	}
	return results
}

func (ck *defaultChecker) GetRunningPeriodicCheckCount() int {
	ck.mtx.Lock()
	defer ck.mtx.Unlock()
	return len(ck.endChans)
}

func newSystemStatus(status AvailabilityStatus, results map[string]CheckStatus, withDetails bool) SystemStatus {
	aggStatus := SystemStatus{Status: status}
	if withDetails {
		aggStatus.Details = &results
	}
	return aggStatus
}

func isCacheExpired(cacheDuration time.Duration, state *CheckState) bool {
	return state.LastCheckedAt == nil || state.LastCheckedAt.Before(time.Now().Add(-cacheDuration))
}

func isPeriodicCheck(check *Check) bool {
	return check.updateInterval > 0
}

func withCheckContext(ctx context.Context, check *Check, f func(checkCtx context.Context)) {
	cancel := func() {}
	if check.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, check.Timeout)
	}
	defer cancel()
	f(ctx)
}

func doCheck(ctx context.Context, check *Check, oldState CheckState) (context.Context, CheckState) {
	state := oldState

	if state.FirstCheckStartedAt.IsZero() {
		state.FirstCheckStartedAt = time.Now().UTC()
	}

	if check.BeforeCheckListener != nil {
		ctx = check.BeforeCheckListener(ctx, check.Name, state)
	}

	state = check.interceptorChain(ctx, check.Name, state)

	if check.StatusListener != nil && oldState.Status != state.Status {
		ctx = check.StatusListener(ctx, state)
	}

	if check.AfterCheckListener != nil {
		check.AfterCheckListener(ctx, state)
	}

	return ctx, state
}

func checkCurrentState(ctx context.Context, check *Check, state CheckState) CheckState {
	now := time.Now().UTC()

	state.Result = executeCheckFunc(ctx, check)
	state.LastCheckedAt = &now

	if state.Result == nil {
		state.ContiguousFails = 0
		state.LastSuccessAt = &now
	} else {
		state.ContiguousFails++
		state.LastFailureAt = &now
	}

	state.Status = evaluateCheckStatus(&state, check.MaxTimeInError, check.MaxContiguousFails)

	return state
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
		Error:     toErrorDesc(state.Result, maxErrMsgLen),
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
	} else if state.Result != nil {
		maxTimeInErrorSinceStartPassed := !state.FirstCheckStartedAt.Add(maxTimeInError).After(time.Now())
		maxTimeInErrorSinceLastSuccessPassed := state.LastSuccessAt == nil || !state.LastSuccessAt.Add(maxTimeInError).After(time.Now())

		timeInErrorThresholdCrossed := maxTimeInErrorSinceStartPassed && maxTimeInErrorSinceLastSuccessPassed
		failCountThresholdCrossed := state.ContiguousFails >= maxFails

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

func newInterceptorChain(check *Check) InterceptorFunc {
	var chain InterceptorFunc = func(ctx context.Context, name string, state CheckState) CheckState {
		return checkCurrentState(ctx, check, state)
	}

	for idx := len(check.Interceptors) - 1; idx >= 0; idx-- {
		intercept := check.Interceptors[idx]
		downstreamChain := chain
		chain = func(ctx context.Context, name string, state CheckState) CheckState {
			return intercept(ctx, name, state, downstreamChain)
		}
	}

	return chain
}
