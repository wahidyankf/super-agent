package behaviour

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A scenario deadline ends when the scenario clock passes its bound, and never before. The unit level relies on this:
// no real timer runs, so a stub that never answers ends its request only when the scenario clock moves past the bound.

// TestAScenarioDeadlineEndsWhenTheClockPassesItsBound checks the bound on the scenario clock: it stays open until the
// clock reaches the bound, then ends with the deadline error, and its Done channel closes.
func TestAScenarioDeadlineEndsWhenTheClockPassesItsBound(t *testing.T) {
	clk := newFakeClock()
	ctx, release := clk.limit(context.Background(), 300*time.Millisecond)
	defer release()
	if ctx.Err() != nil {
		t.Fatalf("the bound ended before the clock moved: %v", ctx.Err())
	}
	if err := clk.Sleep(context.Background(), 200*time.Millisecond); err != nil {
		t.Fatalf("sleep before the bound: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("the bound ended at 200ms of a 300ms limit: %v", ctx.Err())
	}
	if err := clk.Sleep(context.Background(), 100*time.Millisecond); err != nil {
		t.Fatalf("sleep past the bound: %v", err)
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("the bound did not end with the deadline error, got %v", ctx.Err())
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("Done is open after the bound was passed")
	}
}

// TestPassNextLimitMovesTheClockToTheEarliestOpenBound checks the hold seam: passing the next limit moves the clock to the
// earliest open bound and ends that bound only, so a later pass ends the next one.
func TestPassNextLimitMovesTheClockToTheEarliestOpenBound(t *testing.T) {
	clk := newFakeClock()
	start := clk.Now()
	late, releaseLate := clk.limit(context.Background(), 300*time.Millisecond)
	defer releaseLate()
	early, releaseEarly := clk.limit(context.Background(), 100*time.Millisecond)
	defer releaseEarly()
	clk.passNextLimit()
	if !errors.Is(early.Err(), context.DeadlineExceeded) {
		t.Fatalf("the earliest bound is open after its pass, got %v", early.Err())
	}
	if late.Err() != nil {
		t.Fatalf("the later bound ended with the earlier one: %v", late.Err())
	}
	if got := clk.Now(); !got.Equal(start.Add(100 * time.Millisecond)) {
		t.Fatalf("the clock after the pass = %v, want the earliest bound", got.Sub(start))
	}
	clk.passNextLimit()
	if !errors.Is(late.Err(), context.DeadlineExceeded) {
		t.Fatalf("the later bound is open after the second pass, got %v", late.Err())
	}
	clk.passNextLimit()
	if got := clk.Now(); !got.Equal(start.Add(300 * time.Millisecond)) {
		t.Fatalf("a pass with no open bound moved the clock to %v", got.Sub(start))
	}
}

// TestAScenarioDeadlineEndsWithItsParent checks that a parent that ends first ends the bound with the parent's error.
func TestAScenarioDeadlineEndsWithItsParent(t *testing.T) {
	clk := newFakeClock()
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, release := clk.limit(parent, time.Second)
	defer release()
	cancelParent()
	<-ctx.Done()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("the bound did not end with its parent, got %v", ctx.Err())
	}
}

// TestAScenarioDeadlineReleasedByItsOwner checks that the release function ends the bound at once, as cancel does.
func TestAScenarioDeadlineReleasedByItsOwner(t *testing.T) {
	clk := newFakeClock()
	ctx, release := clk.limit(context.Background(), time.Second)
	release()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("release did not end the bound, got %v", ctx.Err())
	}
	if err := clk.Sleep(context.Background(), 2*time.Second); err != nil {
		t.Fatalf("sleep after release: %v", err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("a later clock move changed a released bound: %v", ctx.Err())
	}
}
