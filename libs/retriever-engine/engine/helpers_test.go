package engine_test

import (
	"context"
	_ "embed"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wikiYAML is the spec fixture that every engine test reads. It names example hosts only.
//
//go:embed testdata/wiki.yml
var wikiYAML string

// fixtureKey is the fake API key that the fake credentials port returns. It is not a real key.
const fixtureKey = "fixture-key"

// loadSpecFrom validates a spec document and fails the test when it is refused.
func loadSpecFrom(t *testing.T, doc string) *spec.Spec {
	t.Helper()
	res := spec.Validate([]byte(doc))
	if !res.Valid() {
		t.Fatalf("the fixture spec is refused: %v", res.Err())
	}
	return res.Spec
}

// loadSpec returns the validated fixture spec.
func loadSpec(t *testing.T) *spec.Spec {
	t.Helper()
	return loadSpecFrom(t, wikiYAML)
}

// lowered returns the fixture with one of its lines changed, for a test that lowers a limit.
func lowered(t *testing.T, from, to string) string {
	t.Helper()
	if !strings.Contains(wikiYAML, from) {
		t.Fatalf("the fixture has no %q to lower", from)
	}
	return strings.Replace(wikiYAML, from, to, 1)
}

// replaceIn returns doc with the first occurrence of from changed to to. It fails the test when from is not in doc, so a
// fixture edit can never silently do nothing.
func replaceIn(t *testing.T, doc, from, to string) string {
	t.Helper()
	if !strings.Contains(doc, from) {
		t.Fatalf("the fixture has no %q to change", from)
	}
	return strings.Replace(doc, from, to, 1)
}

// testSource returns a source of the fixture kind, on a documentation host, with the fixture credential name.
func testSource(t *testing.T) domain.Source {
	t.Helper()
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: "example-wiki"})
	if err != nil {
		t.Fatal(err)
	}
	kind, err := kinds.Kind("example-wiki")
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NewSourceID("wiki-main")
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatal(err)
	}
	cred, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(id, kind, tags, endpoint, cred)
}

// fakeCredentials is the credentials port of a test. It returns the fixture key for any name.
type fakeCredentials struct {
	calls int
}

// APIKey returns the fixture key, and counts the calls.
func (f *fakeCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	f.calls++
	return domain.NewApiKey(ref.Name(), fixtureKey)
}

// fakeIdentities is the identity port of a test. It returns a fixed address.
type fakeIdentities struct{}

// Identity returns a fixed documentation address.
func (fakeIdentities) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	return domain.NewIdentity(ref.Name(), "reader@example.test")
}

// fakeClock is the clock of a test. It records each time limit and each wait, and it never reads the real clock or
// starts a timer.
type fakeClock struct {
	mu       sync.Mutex
	limits   []time.Duration
	sleeps   []time.Duration
	sleepErr error
}

// WithTimeout records the limit and returns a cancellable context with no timer.
func (c *fakeClock) WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	c.mu.Lock()
	c.limits = append(c.limits, d)
	c.mu.Unlock()
	return context.WithCancel(parent)
}

// Sleep records the wait and returns at once.
func (c *fakeClock) Sleep(_ context.Context, d time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sleeps = append(c.sleeps, d)
	return c.sleepErr
}

// waits returns the recorded waits.
func (c *fakeClock) waits() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.sleeps...)
}

// limitsSeen returns the recorded time limits.
func (c *fakeClock) limitsSeen() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.limits...)
}

// stubPorts returns the ports of a test that answers from a replay stub. No socket and no clock are involved.
func stubPorts(stub *replay.Stub, clock *fakeClock) engine.Ports {
	return engine.Ports{
		Credentials: &fakeCredentials{},
		Identities:  fakeIdentities{},
		Clock:       clock,
		Transport:   stub,
	}
}

// newStubEngine builds an engine for the spec over a replay stub, and returns both.
func newStubEngine(t *testing.T, s *spec.Spec) (*engine.Engine, *replay.Stub, *fakeClock) {
	t.Helper()
	stub, err := replay.New(s)
	if err != nil {
		t.Fatalf("build the replay stub: %v", err)
	}
	clock := &fakeClock{}
	eng, err := engine.New(s, testSource(t), stubPorts(stub, clock))
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	return eng, stub, clock
}
