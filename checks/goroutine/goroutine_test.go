package goroutine

import (
	"context"
	"testing"
)

func launchGoroutines(count int) func() {
	stop := make(chan struct{}, count)
	for i := 0; i < count; i++ {
		go func() { <-stop }()
	}
	return func() {
		for i := 0; i < count; i++ {
			stop <- struct{}{}
		}
	}
}

func Test_MoreThanThresholdGoroutines(t *testing.T) {
	ctx := context.TODO()
	const threshold uint = 2
	const newGCount int = 5
	check := New(threshold)
	defer launchGoroutines(newGCount)()
	if err := check(ctx); err == nil {
		t.Errorf("health check error is expected as the threshold is %d and the current number of goroutines are at least %d", threshold, newGCount)
	}
}

func Test_LessThanThresholdGoroutines(t *testing.T) {
	ctx := context.TODO()
	const threshold uint = 100000
	check := New(threshold)
	if err := check(ctx); err != nil {
		t.Errorf("health check error is not expected as the threshold for goroutine count is %d. Got error: %v", threshold, err)
	}
}
