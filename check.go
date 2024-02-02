package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type (
	asyncCheck interface {
		evaluateCheckConfig
		name() string
		interceptors() []Interceptor
		start(AsyncCheckInput) chan error
		onStateChange(ctx context.Context, oldState CheckState, newState CheckState)
	}

	checkerConfig struct {
		timeout              time.Duration
		info                 map[string]interface{}
		syncChecks           map[string]*Check
		asyncChecks          map[string]asyncCheck
		cacheTTL             time.Duration
		statusChangeListener func(context.Context, CheckerState)
		interceptors         []Interceptor
		detailsDisabled      bool
		autostartDisabled    bool
		rootCtx              context.Context
	}

	defaultChecker struct {
		started            bool
		startedAndCountMtx sync.RWMutex
		checkStateMtx      sync.RWMutex
		cfg                checkerConfig
		state              CheckerState
		cancel             context.CancelFunc
		asyncCheckCount    int
		asyncWg            sync.WaitGroup
	}

	checkResult struct {
		checkName string
		newState  CheckState
	}

	jsonCheckResult struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp,omitempty"`
		Error     string    `json:"error,omitempty"`
	}

	// Checker is the main checker interface. It provides all health checking logic.
	Checker interface {
		// Start will start all necessary background workers and prepare
		// the checker for further usage.
		Start()
		// Stop stops will stop the checker.
		Stop()
		// Check runs all synchronous (i.e., non-periodic) check functions.
		// It returns the aggregated health status (combined from the results
		// of this executions synchronous checks and the previously reported
		// results of asynchronous/periodic checks. This function expects a
		// context, that may contain deadlines to which will be adhered to.
		// The context will be passed to all downstream calls
		// (such as listeners, component check functions, and interceptors).
		Check(ctx context.Context) CheckerResult
		// GetRunningPeriodicCheckCount returns the number of currently
		// running periodic checks.
		GetRunningPeriodicCheckCount() int
		// IsStarted returns true, if the Checker was started (see Checker.Start)
		// and is currently still running. Returns false otherwise.
		IsStarted() bool
	}

	// CheckerState represents the current state of the Checker.
	CheckerState struct {
		// Status is the aggregated system health status.
		Status AvailabilityStatus
		// CheckState contains the state of all checks.
		CheckState map[string]CheckState
	}

	// CheckState represents the current state of a component check.
	CheckState struct {
		// LastCheckedAt holds the time of when the check was last executed.
		LastCheckedAt time.Time
		// LastCheckedAt holds the last time of when the check did not return an error.
		LastSuccessAt time.Time
		// LastFailureAt holds the last time of when the check did return an error.
		LastFailureAt time.Time
		// FirstCheckStartedAt holds the time of when the first check was started.
		FirstCheckStartedAt time.Time
		// ContiguousFails holds the number of how often the check failed in a row.
		ContiguousFails uint
		// Result holds the error of the last check (nil if successful).
		Result error
		// The current availability status of the check.
		Status AvailabilityStatus
	}

	// CheckerResult holds the aggregated system availability status and
	// detailed information about the individual checks.
	CheckerResult struct {
		// Info contains additional information about this health result.
		Info map[string]interface{} `json:"info,omitempty"`
		// Status is the aggregated system availability status.
		Status AvailabilityStatus `json:"status"`
		// Details contains health information for all checked components.
		Details map[string]CheckResult `json:"details,omitempty"`
	}

	// CheckResult holds a components health information.
	// Attention: This type is converted from/to JSON using a custom
	// marshalling/unmarshalling function (see type jsonCheckResult).
	// This is required because some fields are not converted automatically
	// by the standard json.Marshal/json.Unmarshal functions
	// (such as the error interface). The JSON tags you see here, are
	// just there for the readers' convenience.
	CheckResult struct {
		// Status is the availability status of a component.
		Status AvailabilityStatus `json:"status"`
		// Timestamp holds the time when the check was executed.
		Timestamp time.Time `json:"timestamp,omitempty"`
		// Error contains the check error message, if the check failed.
		Error error `json:"error,omitempty"`
	}

	// Interceptor is factory function that allows creating new instances of
	// a InterceptorFunc. The concept behind Interceptor is similar to the
	// middleware pattern. A InterceptorFunc that is created by calling a
	// Interceptor is expected to forward the function call to the next
	// InterceptorFunc (passed to the Interceptor in parameter 'next').
	// This way, a chain of interceptors is constructed that will eventually
	// invoke of the components health check function. Each interceptor must therefore
	// invoke the 'next' interceptor. If the 'next' InterceptorFunc is not called,
	// the components check health function will never be executed.
	Interceptor func(next InterceptorFunc) InterceptorFunc

	// InterceptorFunc is an interceptor function that intercepts any call to
	// a components health check function.
	InterceptorFunc func(ctx context.Context, checkName string, state CheckState) CheckState

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

// MarshalJSON provides a custom marshaller for the CheckResult type.
func (cr CheckResult) MarshalJSON() ([]byte, error) {
	errorMsg := ""
	if cr.Error != nil {
		errorMsg = cr.Error.Error()
	}

	return json.Marshal(&jsonCheckResult{
		Status:    string(cr.Status),
		Timestamp: cr.Timestamp,
		Error:     errorMsg,
	})
}

func (cr *CheckResult) UnmarshalJSON(data []byte) error {
	var result jsonCheckResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	cr.Status = AvailabilityStatus(result.Status)
	cr.Timestamp = result.Timestamp

	if result.Error != "" {
		cr.Error = errors.New(result.Error)
	}

	return nil
}

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

var (
	CheckTimeoutErr = errors.New("check timed out")
)

func newChecker(cfg checkerConfig) *defaultChecker {
	checkState := map[string]CheckState{}
	for _, check := range cfg.syncChecks {
		checkState[check.Name] = CheckState{Status: StatusUnknown}
	}
	for _, check := range cfg.asyncChecks {
		checkState[check.name()] = CheckState{Status: StatusUnknown}
	}

	checker := defaultChecker{
		cfg:   cfg,
		state: CheckerState{Status: StatusUnknown, CheckState: checkState},
	}

	if !cfg.autostartDisabled {
		checker.Start()
	}

	return &checker
}

// Start implements Checker.Start. Please refer to Checker.Start for more information.
func (ck *defaultChecker) Start() {
	ck.startedAndCountMtx.Lock()

	if !ck.started {
		ctx, cancel := context.WithCancel(ck.cfg.rootCtx)
		ck.cancel = cancel

		ck.started = true
		defer ck.startAsyncChecks(ctx)

		// We run the initial check execution in a separate goroutine so that server startup is not blocked in case of
		// a bad check that runs for a longer period of time.
		go ck.Check(ctx)
	}

	// Attention: We should avoid having this unlock as a deferred function call right after the mutex lock above,
	// since this may cause a deadlock (e.g., startPeriodicChecks requires the mutex lock as well and would block
	// because of the defer order)
	ck.startedAndCountMtx.Unlock()
}

// Stop implements Checker.Stop. Please refer to Checker.Stop for more information.
func (ck *defaultChecker) Stop() {
	ck.cancel()
	ck.asyncWg.Wait()

	ck.startedAndCountMtx.Lock()
	defer ck.startedAndCountMtx.Unlock()

	ck.started = false
	ck.asyncCheckCount = 0
}

// GetRunningPeriodicCheckCount implements Checker.GetRunningPeriodicCheckCount.
// Please refer to Checker.GetRunningPeriodicCheckCount for more information.
//
// Deprecated: Use GetRunningAsyncCheckCount.
func (ck *defaultChecker) GetRunningPeriodicCheckCount() int {
	return ck.GetRunningAsyncCheckCount()
}
func (ck *defaultChecker) GetRunningAsyncCheckCount() int {
	ck.startedAndCountMtx.RLock()
	defer ck.startedAndCountMtx.RUnlock()
	return ck.asyncCheckCount
}

// IsStarted implements Checker.IsStarted. Please refer to Checker.IsStarted for more information.
func (ck *defaultChecker) IsStarted() bool {
	ck.startedAndCountMtx.RLock()
	defer ck.startedAndCountMtx.RUnlock()
	return ck.started
}

// Check implements Checker.Check. Please refer to Checker.Check for more information.
func (ck *defaultChecker) Check(ctx context.Context) CheckerResult {
	ck.checkStateMtx.Lock()
	defer ck.checkStateMtx.Unlock()

	ctx, cancel := context.WithTimeout(ctx, ck.cfg.timeout)
	defer cancel()

	ck.runSynchronousChecks(ctx)

	return ck.mapStateToCheckerResult()
}

func (ck *defaultChecker) runSynchronousChecks(ctx context.Context) {
	var (
		numChecks          = len(ck.cfg.syncChecks)
		numInitiatedChecks = 0
		resChan            = make(chan checkResult, numChecks)
	)

	for _, check := range ck.cfg.syncChecks {
		check := check

		checkState := ck.state.CheckState[check.Name]

		if !isCacheExpired(ck.cfg.cacheTTL, &checkState) {
			continue
		}

		numInitiatedChecks++

		go func() {
			withCheckContext(ctx, check, func(ctx context.Context) {
				_, checkState := executeCheck(ctx, &ck.cfg, check, checkState)
				resChan <- checkResult{check.Name, checkState}
			})
		}()
	}

	results := make([]checkResult, 0, numInitiatedChecks)
	for len(results) < numInitiatedChecks {
		results = append(results, <-resChan)
	}

	ck.updateState(ctx, results...)
}

func (ck *defaultChecker) startAsyncChecks(ctx context.Context) {
	ck.startedAndCountMtx.Lock()
	defer ck.startedAndCountMtx.Unlock()

	// Start streaming checks.
	for _, check := range ck.cfg.asyncChecks {
		check := check

		// ATTENTION: Access to check and ck.state.CheckState is not synchronized here,
		// 	assuming that the accessed values are never changed, such as
		//  - ck.state.CheckState[check.Name]
		//  - check object itself (there will never be a new Check object created for the configured check)
		//	- check.updateInterval (used by isPeriodicCheck)
		//  - check.initialDelay
		// ALSO:
		//  - The check state itself is never synchronized on, since the only place where values can be changed are
		//    within this goroutine.

		ck.asyncCheckCount++
		ck.asyncWg.Add(1)

		go func() {
			defer ck.asyncWg.Done()

			input := newAsyncCheckInput(ctx)
			checksChannel := check.start(input)

			for {
				oldState := ck.getCheckStateWithLock(check.name())
				newState := oldState

				// We copy explicitly to not affect the underlying array of the slices as a side effect.
				// These slices are being passed to this library as configuration parameters, so we don't know how they
				// are being used otherwise in the users program.
				interceptors := make([]Interceptor, 0, len(ck.cfg.interceptors)+len(check.interceptors()))
				interceptors = append(interceptors, ck.cfg.interceptors...)
				interceptors = append(interceptors, check.interceptors()...)

				// ATTENTION: executeCheck is executed with its own copy of the checks
				// 	state (see checkState above). This means that if there is a global status
				//	listener that is configured by the user with health.WithStatusListener,
				//	and that global status listener changes this checks state as long as
				//  executeCheck is running, the modifications made by the global listener
				//  will be lost after the function completes, since we overwrite the state
				//  below using updateState.
				//  This means that global listeners should not change the checks state
				//  or accept losing their updates. This will be the case especially for
				//  long-running checks. Hence, the checkState is read-only for interceptors.

				newState = withInterceptors(interceptors, func(ctx context.Context, _ string, state CheckState) CheckState {
					input.updateStateForNextCheck(ctx, state)
					checkStreamResult, ok := receiveOnCheckChannel(ctx, checksChannel)
					if !ok {
						return state
					}
					newState := createNextCheckState(checkStreamResult, check, oldState)
					newState.FirstCheckStartedAt = input.firstCheckStartedAt
					return newState
				})(ctx, check.name(), newState)

				if ctx.Err() != nil {
					// stop looping if context ended
					return
				}

				ck.updateStateWithLock(ctx, checkResult{check.name(), newState})
				check.onStateChange(ctx, oldState, newState)
			}
		}()
	}
}

func receiveOnCheckChannel(ctx context.Context, checksChannel chan error) (checkStreamResult error, ok bool) {
	select {
	case checkStreamResult, ok := <-checksChannel:
		return checkStreamResult, ok
	case <-ctx.Done():
		return nil, false
	}
}

func (ck *defaultChecker) getCheckStateWithLock(checkName string) CheckState {
	ck.checkStateMtx.RLock()
	defer ck.checkStateMtx.RUnlock()
	checkState := ck.state.CheckState[checkName]
	return checkState
}

func (ck *defaultChecker) updateStateWithLock(ctx context.Context, updates ...checkResult) {
	ck.checkStateMtx.Lock()
	defer ck.checkStateMtx.Unlock()
	ck.updateState(ctx, updates...)
}

func (ck *defaultChecker) updateState(ctx context.Context, updates ...checkResult) {
	for _, update := range updates {
		ck.state.CheckState[update.checkName] = update.newState
	}

	oldStatus := ck.state.Status
	ck.state.Status = aggregateStatus(ck.state.CheckState)

	if oldStatus != ck.state.Status && ck.cfg.statusChangeListener != nil {
		ck.cfg.statusChangeListener(ctx, ck.state)
	}
}

func (ck *defaultChecker) mapStateToCheckerResult() CheckerResult {
	var (
		checkResults map[string]CheckResult
		numChecks    = len(ck.cfg.syncChecks) + len(ck.cfg.asyncChecks)
		status       = ck.state.Status
	)

	if numChecks > 0 && !ck.cfg.detailsDisabled {
		checkResults = make(map[string]CheckResult, numChecks)
		for _, check := range ck.cfg.syncChecks {
			checkState := ck.state.CheckState[check.Name]
			checkResults[check.Name] = CheckResult{
				Status:    checkState.Status,
				Error:     checkState.Result,
				Timestamp: checkState.LastCheckedAt,
			}
		}
		for _, check := range ck.cfg.asyncChecks {
			checkState := ck.state.CheckState[check.name()]
			checkResults[check.name()] = CheckResult{
				Status:    checkState.Status,
				Error:     checkState.Result,
				Timestamp: checkState.LastCheckedAt,
			}
		}
	}

	return CheckerResult{Status: status, Details: checkResults, Info: ck.cfg.info}
}

func isCacheExpired(cacheDuration time.Duration, state *CheckState) bool {
	return state.LastCheckedAt.IsZero() || state.LastCheckedAt.Before(time.Now().Add(-cacheDuration))
}

func waitForStopSignal(ctx context.Context, waitTime time.Duration) bool {
	select {
	case <-time.After(waitTime):
		return false
	case <-ctx.Done():
		return true
	}
}

func withCheckContext(ctx context.Context, check *Check, f func(checkCtx context.Context)) {
	cancel := func() {}
	if check.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, check.Timeout)
	}
	defer cancel()
	f(ctx)
}

func executeCheck(
	ctx context.Context,
	cfg *checkerConfig,
	check *Check,
	oldState CheckState,
) (context.Context, CheckState) {
	newState := oldState

	if newState.FirstCheckStartedAt.IsZero() {
		newState.FirstCheckStartedAt = time.Now().UTC()
	}

	// We copy explicitly to not affect the underlying array of the slices as a side effect.
	// These slices are being passed to this library as configuration parameters, so we don't know how they
	// are being used otherwise in the users program.
	interceptors := make([]Interceptor, 0, len(cfg.interceptors)+len(check.Interceptors))
	interceptors = append(interceptors, cfg.interceptors...)
	interceptors = append(interceptors, check.Interceptors...)

	newState = withInterceptors(interceptors, func(ctx context.Context, _ string, state CheckState) CheckState {
		checkFuncResult := executeCheckFunc(ctx, check)
		return createNextCheckState(checkFuncResult, check, state)
	})(ctx, check.Name, newState)

	check.onStateChange(ctx, oldState, newState)

	return ctx, newState
}

func executeCheckFunc(ctx context.Context, check *Check) error {
	// If this channel is not bounded, we may have a goroutine leak (e.g., when ctx.Done signals first then
	// sending the check result into the channel will block forever).
	res := make(chan error, 1)

	go func() {
		defer func() {
			if !check.DisablePanicRecovery {
				if r := recover(); r != nil {
					// TODO: Provide a configurable panic handler configuration option, so developers can decide
					// 	what to do with panics.
					err, ok := r.(error)
					if ok {
						res <- err
					} else {
						res <- fmt.Errorf("%v", r)
					}
				}
			}
		}()

		res <- check.Check(ctx)
	}()

	select {
	case err := <-res:
		return err
	case <-ctx.Done():
		return CheckTimeoutErr
	}
}

type evaluateCheckConfig interface {
	maxTimeInError() time.Duration
	maxFails() uint
}

func createNextCheckState(result error, check evaluateCheckConfig, state CheckState) CheckState {
	now := time.Now().UTC()

	state.Result = result
	state.LastCheckedAt = now

	if state.Result == nil {
		state.ContiguousFails = 0
		state.LastSuccessAt = now
	} else {
		state.ContiguousFails++
		state.LastFailureAt = now
	}

	state.Status = evaluateCheckStatus(&state, check)

	return state
}

func evaluateCheckStatus(state *CheckState, config evaluateCheckConfig) AvailabilityStatus {
	if state.LastCheckedAt.IsZero() {
		return StatusUnknown
	} else if state.Result != nil {
		maxTimeInError := config.maxTimeInError()
		maxFails := config.maxFails()

		maxTimeInErrorSinceStartPassed := !state.FirstCheckStartedAt.Add(maxTimeInError).After(time.Now())
		maxTimeInErrorSinceLastSuccessPassed := state.LastSuccessAt.IsZero() ||
			!state.LastSuccessAt.Add(maxTimeInError).After(time.Now())

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

func withInterceptors(interceptors []Interceptor, target InterceptorFunc) InterceptorFunc {
	chain := target

	for idx := len(interceptors) - 1; idx >= 0; idx-- {
		chain = interceptors[idx](chain)
	}

	return chain
}
