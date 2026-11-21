package cli

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
)

// TestWithSignalsUsesTheInjectedSource checks the signal seam: withSignals registers its channel with the injected notify
// function, a signal delivered on that channel cancels the context with its cause, and the stop function releases the
// same channel. No test registers a handler with the process. The stop function runs on the signal goroutine, so the test
// receives its argument from a channel and never reads a variable that the goroutine writes.
func TestWithSignalsUsesTheInjectedSource(t *testing.T) {
	var registered chan<- os.Signal
	released := make(chan chan<- os.Signal, 1)
	notify := func(c chan<- os.Signal, sig ...os.Signal) {
		registered = c
		if len(sig) != 2 {
			t.Errorf("notify got %d signals, want the interrupt and SIGTERM", len(sig))
		}
	}
	stop := func(c chan<- os.Signal) { released <- c }

	ctx, release := withSignals(context.Background(), notify, stop)
	if registered == nil {
		t.Fatal("withSignals did not register a channel with the injected notify")
	}
	registered <- syscall.SIGTERM
	<-ctx.Done()
	var c signalCause
	if !errors.As(context.Cause(ctx), &c) || c.sig != syscall.SIGTERM {
		t.Fatalf("the cause = %v, want the SIGTERM cause", context.Cause(ctx))
	}
	release()
	if got := <-released; got != registered {
		t.Fatal("the stop function released a channel other than the one registered")
	}
}

// noSignals is the signal source of the unit tests that run serve or mcp: notify registers nothing, and stop releases
// nothing, so a session never changes the process's signal handlers. The tests that need a signal cancel the context
// with its SignalCause instead.
func noSignals(chan<- os.Signal, ...os.Signal) {}

// noStop releases the channel of noSignals, which registered nothing.
func noStop(chan<- os.Signal) {}

// TestWithSignalsEndsWithItsParentWithoutASignal checks that a parent that ends first ends the context with the parent's
// cause, and that the stop function still releases the channel.
func TestWithSignalsEndsWithItsParentWithoutASignal(t *testing.T) {
	released := make(chan chan<- os.Signal, 1)
	parent, cancel := context.WithCancel(context.Background())
	ctx, release := withSignals(parent, func(chan<- os.Signal, ...os.Signal) {}, func(c chan<- os.Signal) { released <- c })
	cancel()
	<-ctx.Done()
	if context.Cause(ctx) != context.Canceled {
		t.Fatalf("the cause = %v, want context.Canceled", context.Cause(ctx))
	}
	release()
	if <-released == nil {
		t.Fatal("the stop function released no channel")
	}
}
