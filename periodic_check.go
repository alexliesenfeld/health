package health

import "context"

func (check *periodicCheck) name() string {
	return check.Name
}

func (check *periodicCheck) interceptors() []Interceptor {
	return check.Interceptors
}

func (check *periodicCheck) start(input AsyncCheckInput) chan error {
	checkChannel := make(chan error)

	go func() {
		defer close(checkChannel)

		if check.initialDelay > 0 {
			if waitForStopSignal(input.Context(), check.initialDelay) {
				return
			}
		}

		for {
			runInput := input.GetInputForCheck() // This will wait until the input is ready for the next run
			ctx, cancel := context.WithCancel(runInput.Context)
			if check.Timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, check.Timeout)
			}

			// ATTENTION: This function may panic, if panic handling is disabled
			// 	via "check.DisablePanicRecovery".

			checkChannel <- executeCheckFunc(ctx, &check.Check)
			cancel()

			if waitForStopSignal(runInput.Context, check.updateInterval) {
				return
			}
		}
	}()

	return checkChannel
}
