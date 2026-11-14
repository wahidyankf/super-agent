package kiwi

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// sentinelKey is a unique synthetic key. The stub accepts it, and no error, output, or body that a test observes may
// contain it, except the Authorization header the stub records as a match.
const sentinelKey = "sentinel-kiwi-key-7d3a91"

// keyVariable is the variable name the credentials double reports when no key is set.
const keyVariable = "EXAMPLE_KIWI_API_KEY"

// testHost is the endpoint host of the source, which is the Host header of every request. testBase is the origin of the
// in-process stub: its name never resolves, and the in-process transport never dials it.
const (
	testHost = "kiwi.example.test"
	testBase = "https://kiwi-stub.invalid"
)

// epoch is the fixed start of the fake clock.
var epoch = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

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

// fakeClock is a clock the test moves. Sleeping advances it and records the wait, so pacing and Retry-After are asserted
// without any real waiting.
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
	stub  *kiwistub.Stub
	clock *fakeClock
	creds *staticCreds
	gw    *Gateway
}

// newRig builds a gateway over the dataset. The pacing delay is zero unless a test passes WithMinDelay.
func newRig(t *testing.T, d kiwistub.Dataset, opts ...Option) rig {
	t.Helper()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(d))
	clock := newFakeClock()
	creds := &staticCreds{}
	client := &http.Client{Transport: stub.RoundTripper()}
	all := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, opts...)
	gw, err := NewWithBase(testSourceAt(t), testBase, client, creds, all...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return rig{stub: stub, clock: clock, creds: creds, gw: gw}
}

// testSourceAt returns a kiwi source whose endpoint is the test host.
func testSourceAt(t *testing.T) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("kiwi")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://" + testHost)
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef(keyVariable)
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	return domain.NewSource(id, domain.KindKiwi, tags, endpoint, ref)
}

// sampleDataset is the small Kiwi server most tests read: one product with two plans, two cases, and the lookup tables.
func sampleDataset() kiwistub.Dataset {
	return kiwistub.Dataset{
		Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}, {ID: 2, Name: "Empty"}},
		Plans: []kiwistub.Plan{
			{ID: 1, Name: "Smoke", Product: 1, Text: "Run before each release."},
			{ID: 2, Name: "Login", Product: 1, Parent: 1},
		},
		Cases: []kiwistub.Case{
			{ID: 12, Summary: "Login works", Text: "Open the login page.", Plans: []int{2}, Sortkey: 1, Status: 1, Priority: 2, Category: 1},
			{ID: 13, Summary: "Logout works", Text: "Log out.", Plans: []int{2}, Sortkey: 2, Status: 1, Priority: 2, Category: 1},
		},
		Categories: []kiwistub.Category{{ID: 1, Name: "Functional", Product: 1}, {ID: 4, Name: "Regression", Product: 1}, {ID: 5, Name: "Regression", Product: 1}},
		Priorities: []kiwistub.Priority{{ID: 2, Value: "P1"}},
		Statuses:   []kiwistub.Status{{ID: 1, Name: "CONFIRMED"}},
	}
}
