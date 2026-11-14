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

// Sleep waits for d on a real timer, or until ctx ends. When ctx ends first it returns the context's error.
func (realClock) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
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
