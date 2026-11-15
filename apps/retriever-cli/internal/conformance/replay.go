package conformance

import (
	"context"
	"fmt"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// fixtureKey is the fixed fake key the replay engine authenticates with. The
// replay answers from the canned exchanges, so no request leaves the process
// and no real key is ever read.
const fixtureKey = "conformance-fixture-key"

// fixtureIdentity is the fixed fake account address of a basic-auth replay.
const fixtureIdentity = "conformance@example.test"

// fixtureHostOf returns the documentation host the replay source binds
// with: conformance under the first suffix the spec names, so a spec with
// suffixes always binds, or the documentation host of the kind when the spec
// names none. The bundled examples pin the kind host in the absolute URLs
// they expect, so the replay serves the same host the examples were written
// against.
func fixtureHostOf(s *spec.Spec) string {
	if len(s.Endpoint.HostSuffix) != 0 {
		return "conformance" + s.Endpoint.HostSuffix[0]
	}
	return s.Kind + ".example.test"
}

// replayCredentials is the credentials port of a replay: it returns the fixed
// fake key for any variable, and it never reads the environment.
type replayCredentials struct{}

// APIKey returns the fixed fake key.
func (replayCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey(ref.Name(), fixtureKey)
}

// replayIdentities is the identities port of a replay: it returns the fixed
// fake address for any variable.
type replayIdentities struct{}

// Identity returns the fixed fake address.
func (replayIdentities) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	return domain.NewIdentity(ref.Name(), fixtureIdentity)
}

// replayClock is the clock of a replay: it records no time and starts no
// timer, so the replay never reads the real clock.
type replayClock struct{}

// WithTimeout returns a cancellable context with no timer.
func (replayClock) WithTimeout(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// Sleep returns nil at once, without waiting.
func (replayClock) Sleep(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}

// mustKind returns the registered kind of a set, panicking on an unregistered
// name. The set is built from the name just before, so a failure is a broken
// invariant.
func mustKind(kinds domain.KindSet, name string) domain.SourceKind {
	kind, err := kinds.Kind(name)
	if err != nil {
		panic(err)
	}
	return kind
}

// testEngine builds the engine of one validated spec over its replay stub.
// The source is synthetic: the spec's kind, the documentation host, and the
// credential name the spec declares, so basic auth finds its identity variable
// too. A spec the engine refuses cannot replay, and the caller reports it as
// a replay issue with no value.
func testEngine(s *spec.Spec, stub *replay.Stub) (*engine.Engine, error) {
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: s.Kind})
	if err != nil {
		return nil, err
	}
	kind := mustKind(kinds, s.Kind)
	id, err := domain.NewSourceID("conformance")
	if err != nil {
		return nil, err
	}
	endpoint, err := domain.NewEndpoint("https://" + fixtureHostOf(s))
	if err != nil {
		return nil, err
	}
	cred, err := domain.NewCredentialRef(s.Auth.CredentialEnv)
	if err != nil {
		return nil, err
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		return nil, err
	}
	src := domain.NewSource(id, kind, tags, endpoint, cred)
	if s.Auth.Type == "basic" {
		ref, err := domain.NewIdentityRef(s.Auth.IdentityEnv)
		if err != nil {
			return nil, fmt.Errorf("the identity variable is refused: %w", err)
		}
		src = src.WithIdentity(ref)
	}
	return engine.New(s, src, engine.Ports{
		Credentials: replayCredentials{},
		Identities:  replayIdentities{},
		Clock:       replayClock{},
		Transport:   stub,
	})
}

var _ ports.Credentials = replayCredentials{}
var _ ports.Identities = replayIdentities{}
var _ engine.Clock = replayClock{}
