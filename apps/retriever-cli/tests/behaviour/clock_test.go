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
	mu     sync.Mutex
	now    time.Time
	limits []*scenarioDeadline
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
// ended stops the wait with its error, as a real wait does. A time limit whose bound the clock has reached ends too.
func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	c.endReachedLimits()
	return nil
}

// passNextLimit moves the clock to the earliest open bound and ends every bound that the clock reaches. A stub that holds
// a request calls it, so the hold ends at the bound, as a real deadline would end it, and no test waits on real time. It
// does nothing when no bound is open.
func (c *fakeClock) passNextLimit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	var next *scenarioDeadline
	for _, l := range c.limits {
		if l.Err() == nil && (next == nil || l.at.Before(next.at)) {
			next = l
		}
	}
	if next == nil {
		return
	}
	if next.at.After(c.now) {
		c.now = next.at
	}
	c.endReachedLimits()
}

// endReachedLimits ends every time limit whose bound the clock has reached. The caller holds c.mu.
func (c *fakeClock) endReachedLimits() {
	for _, l := range c.limits {
		if !c.now.Before(l.at) {
			l.end(context.DeadlineExceeded)
		}
	}
}

// limit starts a time limit of d on the scenario clock, under parent. The limit ends with the deadline error once a wait
// moves the clock to its bound, with the parent's error when the parent ends first, and with the cancel error when the
// returned release function runs. It starts no timer, so no request ends on real time.
func (c *fakeClock) limit(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	l := &scenarioDeadline{Context: parent, at: c.now.Add(d), done: make(chan struct{})}
	c.limits = append(c.limits, l)
	l.stop = context.AfterFunc(parent, func() { l.end(parent.Err()) })
	return l, func() {
		l.end(context.Canceled)
		l.stop()
	}
}

// scenarioDeadline is a time limit on the scenario clock. It takes its values and its parent's cancellation from the
// parent, and it reports the earlier of its bound and the parent's deadline.
type scenarioDeadline struct {
	context.Context
	at   time.Time
	stop func() bool
	mu   sync.Mutex
	err  error
	done chan struct{}
}

// Deadline returns the earlier of the bound and the parent's deadline.
func (l *scenarioDeadline) Deadline() (time.Time, bool) {
	if pd, ok := l.Context.Deadline(); ok && pd.Before(l.at) {
		return pd, true
	}
	return l.at, true
}

// Done closes when the limit ends.
func (l *scenarioDeadline) Done() <-chan struct{} {
	return l.done
}

// Err returns the error that ended the limit, or nil while it is open.
func (l *scenarioDeadline) Err() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.err
}

// end ends the limit with err. The first end wins, so a later one changes nothing.
func (l *scenarioDeadline) end(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err == nil {
		l.err = err
		close(l.done)
	}
}
