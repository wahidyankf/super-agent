package behaviour

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The in-process levels run a loaded spec through the generic engine, as the composition root does. What the engine may
// reach is the part that differs by level: the unit level answers from the replay stubs in memory, the integration level
// answers from the loopback stub of tests/specstub, and the end-to-end level runs the built binary, which has no runtime
// here. This file holds the parts that every in-process level shares.

// specRuntime is how the engine of one scenario reaches its spec sources, and how many requests reached them. hits counts
// the requests that a source received, and it is nil when no source has a stub to count at.
type specRuntime struct {
	network   engine.Network
	transport http.RoundTripper
	roots     *x509.CertPool
	options   []engine.Option
	hits      func() int
	sent      func() []replay.Request
}

// requests returns how many requests the spec sources of the scenario received.
func (rt specRuntime) requests() int {
	if rt.hits == nil {
		return 0
	}
	return rt.hits()
}

// received returns the requests that the spec sources of the scenario received, in order. A runtime with no source that
// keeps its requests returns none.
func (rt specRuntime) received() []replay.Request {
	if rt.sent == nil {
		return nil
	}
	return rt.sent()
}

// loopbackSpecRuntime starts the loopback stub of one spec and returns the runtime that reaches it. Only the integration
// build sets it, in spec_loopback_integration_test.go, because the stub is built under that tag.
var loopbackSpecRuntime func(t *testing.T, model *spec.Spec) (specRuntime, error)

// replayPool is the unit level's spec source: one replay stub for each loaded spec. A request goes to the first stub that
// holds an exchange for it, and a request that no stub holds is refused.
type replayPool []*replay.Stub

// RoundTrip returns the answer of the first stub that holds an exchange for the request. When none does, it returns the
// refusal of the last stub.
func (p replayPool) RoundTrip(r *http.Request) (*http.Response, error) {
	err := errors.New("no replay stub holds an exchange for this request")
	for _, stub := range p {
		resp, stubErr := stub.RoundTrip(r)
		if stubErr == nil {
			return resp, nil
		}
		err = stubErr
	}
	return nil, err
}

// requests counts the requests that every stub of the pool received.
func (p replayPool) requests() int {
	total := 0
	for _, stub := range p {
		total += len(stub.Requests())
	}
	return total
}

// sent returns the requests that every stub of the pool received, stub by stub, in the order each stub received them.
func (p replayPool) sent() []replay.Request {
	var all []replay.Request
	for _, stub := range p {
		all = append(all, stub.Requests()...)
	}
	return all
}

// refusingNetwork is the network of the unit level: every call is refused, so no spec source reaches a socket.
func refusingNetwork() engine.Network {
	refuse := errors.New("the unit-level network refuses every call")
	return engine.Network{
		Lookup: func(context.Context, string) ([]netip.Addr, error) { return nil, refuse },
		Dial:   func(context.Context, string, string) (net.Conn, error) { return nil, refuse },
	}
}

// engineClock is the engine's clock over the scenario clock. A time limit ends with its parent or its release, and a wait
// moves the scenario clock, so no test waits on a timer.
type engineClock struct {
	clk *fakeClock
}

// WithTimeout returns a context that ends with its parent, or when the release function runs.
func (c engineClock) WithTimeout(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// Sleep is the scenario clock's wait.
func (c engineClock) Sleep(ctx context.Context, d time.Duration) error {
	return c.clk.Sleep(ctx, d)
}

// userSpecsOf returns the user-folder specs of a merged set. The bundled specs of the set never drive the replay
// runtime: only a user spec does, as a user spec alone did before the ports. A user spec never claims a built-in
// kind, because the user-folder loader refuses one, so every spec of a built-in kind is bundled.
func userSpecsOf(set connectorspec.Set) []connectorspec.Spec {
	var out []connectorspec.Spec
	for _, s := range set.Specs {
		if _, err := domain.BuiltinKinds().Kind(s.Model.Kind); err != nil {
			out = append(out, s)
		}
	}
	return out
}

// specRuntimeFor returns the runtime of the spec sources, built once for the scenario from the first set that loads a
// user spec. A set with no user spec needs no runtime, so it is not kept.
func (w *world) specRuntimeFor(set connectorspec.Set) (specRuntime, error) {
	if w.spec != nil {
		return *w.spec, nil
	}
	if len(userSpecsOf(set)) == 0 {
		return specRuntime{network: refusingNetwork()}, nil
	}
	rt, err := w.buildSpecRuntime(set)
	if err != nil {
		return specRuntime{}, err
	}
	w.spec = &rt
	return rt, nil
}

// buildSpecRuntime builds the runtime of the level: replay stubs in memory at the unit level, and the loopback stub of the
// one loaded spec at the integration level. Only the user specs of the set drive it, never the bundled ones.
func (w *world) buildSpecRuntime(set connectorspec.Set) (specRuntime, error) {
	user := userSpecsOf(set)
	switch w.level {
	case levelUnit:
		pool := make(replayPool, 0, len(user))
		for _, s := range user {
			stub, err := replay.New(s.Model)
			if err != nil {
				return specRuntime{}, err
			}
			pool = append(pool, stub)
		}
		return specRuntime{network: refusingNetwork(), transport: pool, hits: pool.requests, sent: pool.sent}, nil
	case levelIntegration:
		if len(user) > 1 {
			return specRuntime{}, errors.New("the integration level serves one spec per scenario")
		}
		if loopbackSpecRuntime == nil {
			return specRuntime{}, errors.New("the integration build holds no loopback spec runtime")
		}
		return loopbackSpecRuntime(w.outer, user[0].Model)
	}
	return specRuntime{}, errors.New("the end-to-end level runs the built binary and has no in-process spec runtime")
}

// specBuilder returns the gateway builder of one spec kind: the generic engine over the validated model, with the ports of
// the scenario. It is the in-process twin of buildSpec in the composition root.
func (w *world) specBuilder(rt specRuntime, model *spec.Spec, creds *envfile.Credentials) registry.Builder {
	return func(src domain.Source) (ports.SourceGateway, error) {
		eng, err := engine.New(model, src, engine.Ports{
			Credentials: creds,
			Identities:  creds,
			Clock:       engineClock{clk: w.clock},
			Network:     rt.network,
			Transport:   rt.transport,
			Roots:       rt.roots,
		}, rt.options...)
		if err != nil {
			return nil, err
		}
		return eng, nil
	}
}
