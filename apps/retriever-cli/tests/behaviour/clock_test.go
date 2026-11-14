package behaviour

import (
	"context"
	"sync"
	"time"
)

// fakeClock is the clock of the in-process levels, the unit and integration levels. Its time moves only when a wait
// ends, so the gap between two requests is exactly the waits the gateway chose between them. The gateways read it through
// their clock and sleep seams, and the stubs stamp each arrival with it. No scenario at those levels waits in real time.
// The end-to-end level runs the built binary and never uses this clock.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// newFakeClock starts the clock at a fixed instant, so every arrival time is the same from one run to the next.
func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)}
}

// Now returns the current fake time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep is the wait seam of the gateways. It moves the clock forward by d and returns at once. A context that has already
// ended stops the wait with its error, as a real wait does.
func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return nil
}
