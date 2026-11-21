package kiwi

import (
	"context"
	"sync"
	"time"
)

// handLimit is a time limit that the test ends by hand. Once expire runs, every limit it started reports the deadline
// error, as a real limit does when its bound passes, so a held request ends on the test's signal and never on real time.
type handLimit struct {
	mu   sync.Mutex
	open []*handDeadline
}

// handDeadline is one limit of a handLimit. It takes its values and cancellation from its parent.
type handDeadline struct {
	context.Context
	done chan struct{}
	once sync.Once
}

// Done closes when the limit ends.
func (d *handDeadline) Done() <-chan struct{} { return d.done }

// Err returns the deadline error once the limit has ended, and nil while it is open.
func (d *handDeadline) Err() error {
	select {
	case <-d.done:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

// start is the limit seam of a gateway: it starts one limit under parent. The duration is not used, because the test ends
// the limit by hand.
func (h *handLimit) start(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
	d := &handDeadline{Context: parent, done: make(chan struct{})}
	h.mu.Lock()
	h.open = append(h.open, d)
	h.mu.Unlock()
	return d, func() { d.once.Do(func() { close(d.done) }) }
}

// expire ends every open limit with the deadline error.
func (h *handLimit) expire() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, d := range h.open {
		d.once.Do(func() { close(d.done) })
	}
	h.open = nil
}
