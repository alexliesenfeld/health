package health

import (
	"context"
	"time"
)

type (
	AsyncCheckRunInput struct {
		Context      context.Context
		CurrentState CheckState
	}
	AsyncCheckInput interface {
		Context() context.Context
		GetInputForCheck() AsyncCheckRunInput
	}

	asyncCheckInput struct {
		context             context.Context
		firstCheckStartedAt time.Time
		currentState        CheckState
		waitForNext         chan struct{}
	}
)

func newAsyncCheckInput(ctx context.Context) *asyncCheckInput {
	return &asyncCheckInput{
		context:     ctx,
		waitForNext: make(chan struct{}),
	}
}

func (input *asyncCheckInput) Context() context.Context {
	return input.context
}

func (input *asyncCheckInput) GetInputForCheck() AsyncCheckRunInput {
	<-input.waitForNext
	if input.firstCheckStartedAt.IsZero() {
		input.firstCheckStartedAt = time.Now().UTC()
	}
	input.waitForNext = make(chan struct{})
	return AsyncCheckRunInput{
		Context:      input.context,
		CurrentState: input.currentState,
	}
}

func (input *asyncCheckInput) updateStateForNextCheck(ctx context.Context, state CheckState) {
	input.context = ctx
	input.currentState = state
	close(input.waitForNext)
}
