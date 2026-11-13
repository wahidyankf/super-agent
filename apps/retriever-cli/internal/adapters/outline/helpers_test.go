package outline

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/sourcecontract"
)

// sentinelKey is a unique dummy key. No error, output, or header that the test
// observes may contain it, except the Authorization header the stub records.
const sentinelKey = "sentinel-outline-key-5c1e8d"

// keyVariable is the variable name the credentials double reports when no key is set.
const keyVariable = "EXAMPLE_OUTLINE_API_KEY"

// epoch is the fixed start of the fake clock.
var epoch = time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)

// staticCreds returns sentinelKey and counts how often the gateway asks for it.
type staticCreds struct {
	mu    sync.Mutex
	calls int
}

func (c *staticCreds) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return domain.NewApiKey(keyVariable, sentinelKey)
}

func (c *staticCreds) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// missingCreds reports a missing key, as the environment adapter does for an unset variable.
type missingCreds struct{}

func (missingCreds) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey(keyVariable, "")
}

// fakeClock is a clock the test moves. Sleeping advances it and records the wait,
// so pacing and Retry-After are asserted without any real waiting.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: epoch}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.sleeps = append(c.sleeps, d)
	c.mu.Unlock()
	return ctx.Err()
}

func (c *fakeClock) recordedSleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.sleeps...)
}

// rig is a gateway wired to an in-process stub, a fake clock, and a counting credentials double.
type rig struct {
	stub  *outlinestub.Stub
	clock *fakeClock
	creds *staticCreds
	gw    *Gateway
}

// newRig builds the gateway with no pacing delay unless a test passes WithMinDelay.
func newRig(t *testing.T, stubOpts []outlinestub.Option, gwOpts ...Option) rig {
	t.Helper()
	clock := newFakeClock()
	creds := &staticCreds{}
	stub := outlinestub.New(append([]outlinestub.Option{outlinestub.WithClock(clock.Now)}, stubOpts...)...)
	client := &http.Client{Transport: stub.RoundTripper()}
	opts := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, gwOpts...)
	gw, err := NewWithBase(testSource(t, testHost), testBase, client, creds, opts...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return rig{stub: stub, clock: clock, creds: creds, gw: gw}
}

// contractGateway builds the Outline gateway for the shared contract suite, served by an in-process stub.
func contractGateway(t *testing.T, w sourcecontract.Wiring) ports.SourceGateway {
	t.Helper()
	stub := outlinestub.New(contractStubOptions(w)...)
	stubFault(stub, w.Fault)
	client := &http.Client{Transport: stub.RoundTripper()}
	gw, err := NewWithBase(testSource(t, testHost), testBase, client, contractCreds(t, w), WithMinDelay(0), WithSleep(noSleep))
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return gw
}

// contractStubOptions maps the suite's dataset onto the stub's data.
func contractStubOptions(w sourcecontract.Wiring) []outlinestub.Option {
	collections := make([]outlinestub.Collection, 0, len(w.Dataset.Collections))
	for _, c := range w.Dataset.Collections {
		collections = append(collections, outlinestub.Collection{ID: c.ID, Name: c.Name})
	}
	documents := make([]outlinestub.Document, 0, len(w.Dataset.Documents))
	for _, d := range w.Dataset.Documents {
		documents = append(documents, outlinestub.Document{
			ID:           d.ID,
			CollectionID: d.CollectionID,
			ParentID:     d.ParentID,
			Title:        d.Title,
			URL:          d.URL,
			Markdown:     d.Markdown,
		})
	}
	return []outlinestub.Option{
		outlinestub.WithCollections(collections...),
		outlinestub.WithDocuments(documents...),
	}
}

// stubFault scripts the stub to answer every request with the suite's fault. The
// missing-key fault is not a server answer, so the stub stays normal for it.
func stubFault(stub *outlinestub.Stub, f sourcecontract.Fault) {
	switch f {
	case sourcecontract.FaultKeyRejected:
		stub.SetFault(outlinestub.Fault{Status: 401})
	case sourcecontract.FaultAccessDenied:
		stub.SetFault(outlinestub.Fault{Status: 403})
	case sourcecontract.FaultUpstream:
		stub.SetFault(outlinestub.Fault{Status: 500})
	}
}

// contractCreds returns a missing-key credentials double for FaultMissingKey, and a valid one otherwise.
func contractCreds(t *testing.T, w sourcecontract.Wiring) ports.Credentials {
	t.Helper()
	if w.Fault == sourcecontract.FaultMissingKey {
		return missingCreds{}
	}
	return &staticCreds{}
}

// noSleep is a wait that returns at once, for the contract runs that do not test pacing.
func noSleep(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}
