package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRealClockSleepEndsOnItsTimerOrItsContext checks Sleep with its timer replaced: a timer that has fired ends the
// sleep with no error, and a context that has ended ends it with the context's error. No real timer starts. The real
// timer is checked at the integration level.
func TestRealClockSleepEndsOnItsTimerOrItsContext(t *testing.T) {
	saved := newTimer
	t.Cleanup(func() { newTimer = saved })

	fired := make(chan time.Time, 1)
	fired <- time.Time{}
	newTimer = func(time.Duration) (<-chan time.Time, func()) { return fired, func() {} }
	if err := (realClock{}).Sleep(context.Background(), time.Hour); err != nil {
		t.Fatalf("Sleep after its timer fired = %v, want nil", err)
	}

	pending := make(chan time.Time)
	newTimer = func(time.Duration) (<-chan time.Time, func()) { return pending, func() {} }
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (realClock{}).Sleep(ended, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("Sleep after its context ended = %v, want context.Canceled", err)
	}
}

// TestRealClockWithTimeoutEndsAContextAtItsBound checks the bound of WithTimeout with a zero duration: the context is
// already past its deadline, so it ends at once and starts no timer.
func TestRealClockWithTimeoutEndsAContextAtItsBound(t *testing.T) {
	ctx, release := (realClock{}).WithTimeout(context.Background(), 0)
	defer release()
	if err := ctx.Err(); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("context error = %v, want context.DeadlineExceeded", err)
	}
}
