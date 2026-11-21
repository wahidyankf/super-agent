package unit_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wireNetOps is a read and a write of one document, on the example host.
const wireNetOps = `  - id: example-wiki.docs.get
    summary: Read one document.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
        expect:
          title: First document
  - id: example-wiki.docs.create
    summary: Create one document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Name
        in: body
        required: true
    body:
      title: "{{title}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: First document
        response:
          status: 201
          body:
            id: doc-1
        expect:
          id: doc-1
`

// wireNetwork is the network port of a binding. Lookup answers from a table by host, and Dial records each address it
// is asked to reach and returns the dial error of the binding. It makes no connection and asks no resolver.
type wireNetwork struct {
	mu        sync.Mutex
	answers   map[string][]netip.Addr
	lookupErr error
	dialErr   error
	lookups   []string
	dials     []string
}

// network returns the port that the engine receives.
func (n *wireNetwork) network() engine.Network {
	return engine.Network{Lookup: n.lookup, Dial: n.dial}
}

// lookup records the host and returns the answer of the table, or the lookup error of the binding.
func (n *wireNetwork) lookup(_ context.Context, host string) ([]netip.Addr, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lookups = append(n.lookups, host)
	if n.lookupErr != nil {
		return nil, n.lookupErr
	}
	return n.answers[host], nil
}

// dial records the address and returns the dial error of the binding.
func (n *wireNetwork) dial(_ context.Context, _, address string) (net.Conn, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.dials = append(n.dials, address)
	return nil, n.dialErr
}

// lookupCount returns how many times the resolver was asked.
func (n *wireNetwork) lookupCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.lookups)
}

// dialed returns a copy of the addresses that were dialed.
func (n *wireNetwork) dialed() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.dials...)
}

// ipv4Text returns the dotted form of the IPv4 address with the four octets given, so a test source line holds no address
// literal.
func ipv4Text(a, b, c, d byte) string {
	return netip.AddrFrom4([4]byte{a, b, c, d}).String()
}

// wireAnswers returns a table that answers the example host with the given addresses.
func wireAnswers(addrs ...string) map[string][]netip.Addr {
	var out []netip.Addr
	for _, a := range addrs {
		out = append(out, netip.MustParseAddr(a))
	}
	return map[string][]netip.Addr{"docs.example.test": out}
}

// wireNetEngine builds the engine of a spec over the network port of a binding, with no transport of its own, so every
// request passes the network guard. It fails the test when the engine is refused.
func wireNetEngine(t *testing.T, model *spec.Spec, n *wireNetwork, opts ...engine.Option) *engine.Engine {
	t.Helper()
	ports := engine.Ports{
		Credentials: &rebindCredentials{},
		Identities:  rebindIdentities{},
		Clock:       &rebindClock{},
		Network:     n.network(),
	}
	eng, err := engine.New(model, rebindSource(t), ports, opts...)
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	return eng
}

// wireNetDocArgs are the arguments of the read of doc-1.
func wireNetDocArgs() domain.Args {
	return domain.NewArgs(map[string]any{"id": "doc-1"})
}

// TestNetworkGuardRefusesAnIPHostBeforeAnyLookupOrDial sends a request to an IP literal, with the catalog setting off and on.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A host that is an IP address is refused before any lookup or dial, with or without the catalog setting
func TestNetworkGuardRefusesAnIPHostBeforeAnyLookupOrDial(t *testing.T) {
	for _, allow := range []bool{false, true} {
		n := &wireNetwork{answers: wireAnswers(ipv4Text(10, 0, 0, 1))}
		tr := engine.GuardedTransport(n.network(), allow, nil)
		req, err := http.NewRequest(http.MethodGet, "https://"+ipv4Text(10, 0, 0, 1)+"/v1/docs/doc-1", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tr.RoundTrip(req)
		if !errors.Is(err, engine.ErrNetworkRefused) {
			t.Fatalf("allow=%v: err = %v, want the network refusal", allow, err)
		}
		if n.lookupCount() != 0 || len(n.dialed()) != 0 {
			t.Fatalf("allow=%v: the IP host reached the resolver or the dialer", allow)
		}
	}
}

// TestNetworkGuardNeverConnectsToALocalName sends requests to localhost and a subdomain of it, with the setting off and on.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A local name is never connected to, with or without the catalog setting
func TestNetworkGuardNeverConnectsToALocalName(t *testing.T) {
	for _, host := range []string{"localhost", "api.localhost"} {
		for _, allow := range []bool{false, true} {
			n := &wireNetwork{answers: map[string][]netip.Addr{host: {netip.MustParseAddr("127.0.0.1")}}}
			tr := engine.GuardedTransport(n.network(), allow, nil)
			req, err := http.NewRequest(http.MethodGet, "https://"+host+"/v1/docs/doc-1", nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = tr.RoundTrip(req)
			if !errors.Is(err, engine.ErrNetworkRefused) {
				t.Fatalf("%s allow=%v: err = %v, want the network refusal", host, allow, err)
			}
			if len(n.dialed()) != 0 {
				t.Fatalf("%s allow=%v: a connection was made", host, allow)
			}
		}
	}
}

// TestPrivateResolverAnswerFailsBeforeAnyDial reads a host that resolves to a private address.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A resolver answer in a private range fails the call before any dial, so the host never sees the key
func TestPrivateResolverAnswerFailsBeforeAnyDial(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers(ipv4Text(10, 0, 0, 1))}
	eng := wireNetEngine(t, model, n)
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if len(n.dialed()) != 0 {
		t.Fatalf("the call dialed %v before the refusal, so the host could see the key", n.dialed())
	}
}

// TestOnePrivateAddressRefusesTheWholeHost reads a host that resolves to a public address and a private address.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A resolver answer with one private address refuses the whole host
func TestOnePrivateAddressRefusesTheWholeHost(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers("93.184.216.34", ipv4Text(10, 0, 0, 1))}
	eng := wireNetEngine(t, model, n)
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if len(n.dialed()) != 0 {
		t.Fatalf("a dial was made to %v, want none", n.dialed())
	}
}

// TestDialGoesToThePublicAddressOnThePortOfTheRequest reads a host that resolves to one public address.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: The dial goes to the public address the guard checked, on the port of the request
func TestDialGoesToThePublicAddressOnThePortOfTheRequest(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers("93.184.216.34"), dialErr: errors.New("the fixture refuses the dial")}
	eng := wireNetEngine(t, model, n)
	if _, err := eng.Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs()); err == nil {
		t.Fatal("the read succeeded without a connection")
	}
	if got := n.dialed(); len(got) != 1 || got[0] != "93.184.216.34:443" {
		t.Fatalf("dialed %v, want the checked address on port 443", got)
	}
}

// TestCatalogSettingAllowsLoopbackOnly reads three hosts that resolve to loopback, unspecified, and multicast addresses,
// with the catalog setting allowing private networks.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: The catalog setting allows a loopback address and still refuses an unspecified or a multicast address
func TestCatalogSettingAllowsLoopbackOnly(t *testing.T) {
	model := rebindSpec(t, wireNetOps)

	loop := &wireNetwork{answers: wireAnswers("127.0.0.1"), dialErr: errors.New("the fixture refuses the dial")}
	_, _ = wireNetEngine(t, model, loop, engine.AllowPrivateNetwork()).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if got := loop.dialed(); len(got) != 1 || got[0] != "127.0.0.1:443" {
		t.Fatalf("loopback dialed %v, want 127.0.0.1:443", got)
	}

	for _, addr := range []string{"0.0.0.0", "224.0.0.1"} {
		n := &wireNetwork{answers: wireAnswers(addr)}
		_, err := wireNetEngine(t, model, n, engine.AllowPrivateNetwork()).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
		if !errors.Is(err, engine.ErrNetworkRefused) {
			t.Fatalf("%s: err = %v, want the network refusal even with the setting", addr, err)
		}
		if len(n.dialed()) != 0 {
			t.Fatalf("%s was dialed", addr)
		}
	}
}

// TestAddressOfAPrivateClassIsRefusedWithTheSettingOff reads a host that resolves to an address of a private class, with the
// catalog setting off.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: An address of the class <class> is refused unless the catalog setting allows private networks
func TestAddressOfAPrivateClassIsRefusedWithTheSettingOff(t *testing.T) {
	cases := []struct {
		class string
		addr  string
	}{
		{"private range 10.0.0.0/8", ipv4Text(10, 0, 0, 1)},
		{"private range 192.168.0.0/16", ipv4Text(192, 168, 1, 1)},
		{"loopback", "127.0.0.1"},
		{"link-local", "169.254.10.10"},
		{"shared address space", ipv4Text(100, 64, 1, 1)},
		{"unique-local IPv6", "fd00::1"},
	}
	model := rebindSpec(t, wireNetOps)
	for _, c := range cases {
		t.Run(c.class, func(t *testing.T) {
			n := &wireNetwork{answers: wireAnswers(c.addr)}
			_, err := wireNetEngine(t, model, n).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
			if !errors.Is(err, engine.ErrNetworkRefused) {
				t.Fatalf("err = %v, want the network refusal", err)
			}
			if len(n.dialed()) != 0 {
				t.Fatalf("%s was dialed", c.addr)
			}
		})
	}
}

// TestDialErrorIsReturnedAndIsNotARefusal reads a host whose checked public address fails to dial.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A dial error to a checked address is returned as the dial error and is not a refusal
func TestDialErrorIsReturnedAndIsNotARefusal(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers("93.184.216.34"), dialErr: errors.New("the fixture dial failed")}
	_, err := wireNetEngine(t, model, n).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if err == nil || !strings.Contains(err.Error(), "the fixture dial failed") {
		t.Fatalf("err = %v, want the dial error", err)
	}
	if errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatal("a dial error was reported as a refusal")
	}
}

// TestHostWithoutAnAddressIsUnreachable reads a host whose lookup fails, and then one whose answer is empty.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A host that gives no address is unreachable
func TestHostWithoutAnAddressIsUnreachable(t *testing.T) {
	cases := []struct {
		problem string
		n       *wireNetwork
	}{
		{"fails the lookup", &wireNetwork{lookupErr: errors.New("the fixture lookup failed")}},
		{"returns an empty answer", &wireNetwork{}},
	}
	model := rebindSpec(t, wireNetOps)
	for _, c := range cases {
		t.Run(c.problem, func(t *testing.T) {
			_, err := wireNetEngine(t, model, c.n).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
			if !errors.Is(err, domain.ErrUnreachable) {
				t.Fatalf("err = %v, want ErrUnreachable", err)
			}
		})
	}
}

// TestAddressWithoutAPortIsRefusedBeforeTheResolver dials a target that holds a host name and no port.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: An address without a port is refused before the resolver is asked
func TestAddressWithoutAPortIsRefusedBeforeTheResolver(t *testing.T) {
	n := &wireNetwork{answers: wireAnswers("93.184.216.34")}
	tr := engine.GuardedTransport(n.network(), false, nil)
	_, err := tr.DialContext(context.Background(), "tcp", "docs.example.test")
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if n.lookupCount() != 0 {
		t.Fatal("the resolver was asked for an address without a port")
	}
}

// TestWriteWhoseSourceNameIsPrivateIsRefusedBeforeTheDialer writes to a source name that resolves to a private address.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A write whose source name resolves to a private address is refused before the dialer runs
func TestWriteWhoseSourceNameIsPrivateIsRefusedBeforeTheDialer(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers(ipv4Text(10, 0, 0, 1))}
	eng := wireNetEngine(t, model, n)
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": "First document"}))
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a refusal before the dialer was reported as an unknown outcome")
	}
	if len(n.dialed()) != 0 {
		t.Fatal("the dialer ran for a refused write")
	}
}

// TestTransportSendsEveryDialThroughTheGuard dials a private address through the transport the engine builds.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: The transport the engine builds sends every dial through the guard
func TestTransportSendsEveryDialThroughTheGuard(t *testing.T) {
	n := &wireNetwork{answers: wireAnswers(ipv4Text(10, 0, 0, 1))}
	tr := engine.GuardedTransport(n.network(), false, nil)
	_, err := tr.DialContext(context.Background(), "tcp", "docs.example.test:443")
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if len(n.dialed()) != 0 {
		t.Fatal("the guard let a dial reach the dialer")
	}
}

// TestDialThatReportsTheDeadlineIsATimeout makes the dial report the deadline at once.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A dial that reports the deadline is reported as a timeout with the timeout exit status
func TestDialThatReportsTheDeadlineIsATimeout(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	n := &wireNetwork{answers: wireAnswers("93.184.216.34"), dialErr: context.DeadlineExceeded}
	_, err := wireNetEngine(t, model, n).Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("err = %v, want ErrTimedOut", err)
	}
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("exit status = %d, want %d", got, engine.ExitTimedOut)
	}
}

// TestDroppedConnectionDuringTheBodyIsUnreachableWithoutTheKey drops the connection while the reply body is read, and the
// error text carries the key.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A connection that drops while the reply body is read is reported as unreachable with no key in the text
func TestDroppedConnectionDuringTheBodyIsUnreachableWithoutTheKey(t *testing.T) {
	model := rebindSpec(t, wireNetOps)
	rt := rebindReplies(rebindReply{
		status:  http.StatusOK,
		header:  http.Header{"Content-Type": {"application/json"}},
		body:    `{"title":"Fir`,
		bodyErr: errors.New("the connection dropped while the key " + rebindKey + " was in flight"),
	})
	eng := rebindEngine(t, model, rt, &rebindClock{})
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", wireNetDocArgs())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if strings.Contains(err.Error(), rebindKey) {
		t.Fatalf("the error text holds the key: %q", err.Error())
	}
}

// TestLoopbackIsDialedOnlyWithTheCatalogSetting checks a loopback target against the guard with and without the setting.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A loopback address is never dialed without the catalog setting and is dialed with it
func TestLoopbackIsDialedOnlyWithTheCatalogSetting(t *testing.T) {
	off := &wireNetwork{answers: wireAnswers("127.0.0.1")}
	if _, err := engine.GuardedTransport(off.network(), false, nil).DialContext(context.Background(), "tcp", "docs.example.test:443"); !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("without the setting: err = %v, want the network refusal", err)
	}
	if len(off.dialed()) != 0 {
		t.Fatal("a loopback address was dialed without the catalog setting")
	}

	on := &wireNetwork{answers: wireAnswers("127.0.0.1"), dialErr: errors.New("the fixture refuses the dial")}
	_, _ = engine.GuardedTransport(on.network(), true, nil).DialContext(context.Background(), "tcp", "docs.example.test:443")
	if got := on.dialed(); len(got) != 1 || got[0] != "127.0.0.1:443" {
		t.Fatalf("with the setting dialed %v, want the checked address 127.0.0.1:443", got)
	}
}
