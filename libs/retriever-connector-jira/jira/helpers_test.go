package jira

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/jirastub"
)

// testHost is the endpoint host of the test source. It is the Host header of every request and the TLS name the
// certificate must match. testBase is the origin the in-process stub answers: a .invalid name that never resolves, so
// no unit test can reach a real host.
const (
	testHost = "jira.example.test"
	testBase = "https://jira-stub.invalid"
)

// The synthetic credentials. The stub accepts this pair and no other. The token is never a real key, and the e-mail
// address is a made-up one. No test, error, or output may contain either value, except the Basic header the stub checks.
const (
	sentinelToken = "sentinel-jira-token-4f8a2c"
	sentinelEmail = "sentinel.jira.user@example.com"
	keyVariable   = "EXAMPLE_JIRA_API_TOKEN"
	idVariable    = "EXAMPLE_JIRA_EMAIL"
)

// epoch is the fixed start of the fake clock.
var epoch = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

// testSource returns a jira read source whose endpoint is testHost, with its key and identity names.
func testSource(t *testing.T) domain.Source {
	t.Helper()
	return testSourceAt(t, testHost)
}

// testSourceAt returns a jira source whose endpoint is host.
func testSourceAt(t *testing.T, host string) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("jira")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	endpoint, err := domain.NewEndpoint("https://" + host)
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	key, err := domain.NewCredentialRef(keyVariable)
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	identity, err := domain.NewIdentityRef(idVariable)
	if err != nil {
		t.Fatalf("NewIdentityRef: %v", err)
	}
	return domain.NewSource(id, domain.KindJira, tags, endpoint, key).WithIdentity(identity)
}

// fakeSecrets is the credentials double of the gateway. It records the order of its reads, so a test can prove the key
// is read first and the identity second, for each request.
type fakeSecrets struct {
	mu    sync.Mutex
	key   string
	email string
	order []string
}

// newSecrets returns the synthetic credentials the stub accepts.
func newSecrets() *fakeSecrets {
	return &fakeSecrets{key: sentinelToken, email: sentinelEmail}
}

// APIKey returns the key. An empty key is a missing variable, as the environment reader reports it.
func (f *fakeSecrets) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, "key")
	return domain.NewApiKey(keyVariable, f.key)
}

// Identity returns the e-mail address, with the same missing and invalid rules as the environment reader.
func (f *fakeSecrets) Identity(domain.IdentityRef) (domain.Identity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.order = append(f.order, "identity")
	return domain.NewIdentity(idVariable, f.email)
}

// reads returns the names of the reads so far, in order.
func (f *fakeSecrets) reads() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.order...)
}

// fakeClock is a clock that the test moves. Sleeping advances it and records the wait, so pacing and Retry-After are
// asserted without any real waiting.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: epoch}
}

// Now returns the fake time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep advances the fake time by d and records the wait.
func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.sleeps = append(c.sleeps, d)
	c.mu.Unlock()
	return ctx.Err()
}

// recordedSleeps returns every wait so far.
func (c *fakeClock) recordedSleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.sleeps...)
}

// rig is a gateway wired to an in-process stub, a fake clock, and the credentials double.
type rig struct {
	stub    *jirastub.Stub
	clock   *fakeClock
	secrets *fakeSecrets
	gw      *Gateway
}

// newRig builds the gateway with no pacing delay unless a test passes WithMinDelay. The stub answers through its
// RoundTripper, so no socket is opened.
func newRig(t *testing.T, gwOpts ...Option) rig {
	t.Helper()
	clock := newFakeClock()
	secrets := newSecrets()
	stub := jirastub.New(sentinelEmail, sentinelToken)
	client := &http.Client{Transport: stub.RoundTripper()}
	opts := append([]Option{WithMinDelay(0), WithClock(clock.Now), WithSleep(clock.Sleep)}, gwOpts...)
	gw, err := NewWithBase(testSource(t), testBase, client, secrets, opts...)
	if err != nil {
		t.Fatalf("NewWithBase: %v", err)
	}
	return rig{stub: stub, clock: clock, secrets: secrets, gw: gw}
}

// noSleep is a wait that returns at once, for tests that do not measure pacing.
func noSleep(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}
