//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// TestTimingAndRetries_RealStallTimesOutWithOneRequest binds a read whose request stalls on a real connection under a one-second
// spec limit to the timeout exit status, with exactly one request received by the server. The server answers only when the
// test ends, so the wait is real.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A stalled read stops at its time limit with the timeout exit status and sends no second request
func TestTimingAndRetries_RealStallTimesOutWithOneRequest(t *testing.T) {
	release := make(chan struct{})
	lb := reintgStart(t, func(http.ResponseWriter, *http.Request) {
		<-release
	})
	t.Cleanup(func() { close(release) })
	s := reintgSpecFrom(t, reintgTimeoutHeader(), reintgObjectOp)
	eng := reintgEngine(t, s, lb, engine.AllowPrivateNetwork())
	_, err := eng.Read(context.Background(), "example-wiki.docs.one", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("the error is %v with exit status %d, want the timeout exit status 124", err, got)
	}
	if n := lb.hits.Load(); n != 1 {
		t.Fatalf("the server received %d requests, want exactly one", n)
	}
}
