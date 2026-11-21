package kiwi

import (
	"context"
	"testing"
	"time"
)

// TestWithLimitStartsTheLimitOfEachRequest checks the limit seam: every request runs under the limit the gateway was
// given, for the gateway's time limit, so a test can start the limit on its own clock.
func TestWithLimitStartsTheLimitOfEachRequest(t *testing.T) {
	var starts []time.Duration
	limit := func(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
		starts = append(starts, d)
		return context.WithCancel(parent)
	}
	r := newRig(t, sampleDataset(), WithTimeout(7*time.Second), WithLimit(limit))
	if _, err := r.gw.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections with a limit seam: %v", err)
	}
	if len(starts) != 1 || starts[0] != 7*time.Second {
		t.Fatalf("the limit started %v, want one start of 7s", starts)
	}
}
