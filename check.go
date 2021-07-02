package health

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type (
	healthCheckConfig struct {
		middleware               []Middleware
		timeout                  time.Duration
		checks                   []*Check
		maxErrMsgLen             uint
		cacheDuration            time.Duration
		manualPeriodicCheckStart bool
	}

	checkState struct {
		startedAt        time.Time
		lastCheckedAt    time.Time
		lastSuccessAt    time.Time
		lastResult       error
		consecutiveFails uint
	}

	checker struct {
		mtx      sync.Mutex
		cfg      healthCheckConfig
		state    map[string]checkState
		endChans []chan struct{}
	}

	checkResult struct {
		check    Check
		newState checkState
	}

	checkStatus struct {
		Status    availabilityStatus     `json:"status"`
		Timestamp time.Time              `json:"timestamp,omitempty"`
		Error     *string                `json:"error,omitempty"`
		Checks    map[string]checkStatus `json:"checks,omitempty"`
	}

	availabilityStatus uint
)

const (
	statusUp availabilityStatus = iota
	statusWarn
	statusUnknown
	statusDown
)

func (s availabilityStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal([...]string{"UP", "WARN", "UNKNOWN", "DOWN"}[s])
}

func newChecker(cfg healthCheckConfig) checker {
	state := map[string]checkState{}
	for _, check := range cfg.checks {
		state[check.Name] = checkState{
			startedAt: time.Now(),
		}
	}
	return checker{sync.Mutex{}, cfg, state, []chan struct{}{}}
}

func (ck *checker) StartPeriodicChecks() {
	ck.mtx.Lock()
	defer ck.mtx.Unlock()

	for _, check := range ck.cfg.checks {
		if isPeriodicCheck(check) {
			endChan := make(chan struct{}, 1)
			ck.endChans = append(ck.endChans, endChan)
			go func(check Check, currentState checkState) {
			loop:
				for {
					currentState = doCheck(context.Background(), check, currentState).newState
					ck.mtx.Lock()
					ck.state[check.Name] = currentState
					ck.mtx.Unlock()
					select {
					case <-time.After(check.refreshInterval):
					case <-endChan:
						break loop
					}
				}
				close(endChan)
			}(*check, ck.state[check.Name])
		}
	}
}

func (ck *checker) StopPeriodicChecks() {
	ck.mtx.Lock()
	defer ck.mtx.Unlock()
	for _, endChan := range ck.endChans {
		endChan <- struct{}{}
	}
	ck.endChans = []chan struct{}{}
}

func (ck *checker) Check(ctx context.Context) checkStatus {
	ck.mtx.Lock()
	defer ck.mtx.Unlock()

	var (
		numChecks     = len(ck.cfg.checks)
		resChan       = make(chan checkResult, numChecks)
		results       = map[string]checkStatus{}
		cacheDuration = ck.cfg.cacheDuration
		maxErrMsgLen  = ck.cfg.maxErrMsgLen
		numPendingRes = 0
	)

	for _, c := range ck.cfg.checks {
		state := ck.state[c.Name]
		if !isPeriodicCheck(c) && isCacheExpired(cacheDuration, &state) {
			numPendingRes++
			go func(ctx context.Context, check Check, state checkState) {
				resChan <- doCheck(ctx, check, state)
			}(ctx, *c, state)
		} else {
			results[c.Name] = newCheckStatus(c, &state, maxErrMsgLen)
		}
	}

	for numPendingRes > 0 {
		res := <-resChan
		ck.state[res.check.Name] = res.newState
		results[res.check.Name] = newCheckStatus(&res.check, &res.newState, maxErrMsgLen)
		numPendingRes--
	}

	return aggregateStatus(results)
}

func isCacheExpired(cacheDuration time.Duration, state *checkState) bool {
	return state.lastCheckedAt.Before(time.Now().Add(-cacheDuration))
}

func isPeriodicCheck(check *Check) bool {
	return check.refreshInterval > 0
}

func doCheck(ctx context.Context, check Check, state checkState) checkResult {
	ctx, cancel := ctx, func() {}
	if check.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, check.Timeout)
	}
	defer cancel()

	state.lastResult = executeCheckFunc(ctx, &check)
	state.lastCheckedAt = time.Now().UTC()
	if state.lastResult == nil {
		state.consecutiveFails = 0
		state.lastSuccessAt = state.lastCheckedAt
	} else {
		state.consecutiveFails++
	}

	return checkResult{check, state}
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

func newCheckStatus(check *Check, state *checkState, maxErrMsgLen uint) checkStatus {
	return checkStatus{
		Status:    evaluateAvailabilityStatus(state, check.FailureTolerance, check.FailureToleranceThreshold),
		Error:     toErrorDesc(state.lastResult, maxErrMsgLen),
		Timestamp: state.lastCheckedAt,
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

func evaluateAvailabilityStatus(state *checkState, maxTimeInError time.Duration, maxFails uint) availabilityStatus {
	if state.lastCheckedAt.IsZero() {
		return statusUnknown
	} else if state.lastResult != nil {
		maxTimeInErrorSinceStartPassed := state.startedAt.Add(maxTimeInError).Before(time.Now())
		maxTimeInErrorSinceLastSuccessPassed := state.lastSuccessAt.Add(maxTimeInError).Before(time.Now())

		timeInErrorThresholdCrossed := maxTimeInErrorSinceStartPassed && maxTimeInErrorSinceLastSuccessPassed
		failCountThresholdCrossed := state.consecutiveFails >= maxFails

		if failCountThresholdCrossed && timeInErrorThresholdCrossed {
			return statusDown
		}
		return statusWarn
	} else {
		return statusUp
	}
}

func aggregateStatus(results map[string]checkStatus) checkStatus {
	ts := time.Now()
	status := statusUp

	for _, result := range results {
		if result.Timestamp.Before(ts) {
			ts = result.Timestamp
		}
		if result.Status > status {
			status = result.Status
		}
	}

	return checkStatus{
		status,
		ts,
		nil,
		results,
	}
}
