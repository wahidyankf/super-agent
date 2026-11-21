package main

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// realClock is the clock of the process. The engine uses it for the time limit of each request and for the waits between
// two tries of a rate-limited read.
type realClock struct{}

// WithTimeout returns a context that ends after d, and the function that releases it.
func (realClock) WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}

// newTimer starts the real timer that Sleep waits on. It returns the channel that fires when d has passed, and the
// function that stops the timer. A unit test replaces it, so the sleep is checked without starting a real timer.
var newTimer = func(d time.Duration) (<-chan time.Time, func()) {
	timer := time.NewTimer(d)
	return timer.C, func() { timer.Stop() }
}

// Sleep waits for d on a real timer, or until ctx ends. When ctx ends first it returns the context's error.
func (realClock) Sleep(ctx context.Context, d time.Duration) error {
	fired, stop := newTimer(d)
	defer stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-fired:
		return nil
	}
}

// realNetwork returns the system resolver and the standard dialer. The engine checks every address that the resolver
// returns before it dials, so these two calls reach the network and nothing else.
func realNetwork() engine.Network {
	var dialer net.Dialer
	return engine.Network{
		Lookup: func(ctx context.Context, host string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		},
		Dial: dialer.DialContext,
	}
}
