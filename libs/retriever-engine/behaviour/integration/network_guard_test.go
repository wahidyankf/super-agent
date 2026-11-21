//go:build integration

package integration_test

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
)

// loopCountingNetwork wraps the network of a server and records each resolver call and each dial, so a binding can see
// that a refused connection never reached the resolver or the dialer.
type loopCountingNetwork struct {
	mu      sync.Mutex
	lookups int
	dials   []string
	base    engine.Network
}

// network returns the counting port, which forwards to the base port.
func (c *loopCountingNetwork) network() engine.Network {
	return engine.Network{
		Lookup: func(ctx context.Context, host string) ([]netip.Addr, error) {
			c.mu.Lock()
			c.lookups++
			c.mu.Unlock()
			return c.base.Lookup(ctx, host)
		},
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			c.mu.Lock()
			c.dials = append(c.dials, address)
			c.mu.Unlock()
			return c.base.Dial(ctx, network, address)
		},
	}
}

// counts returns the number of resolver calls and the dialed addresses.
func (c *loopCountingNetwork) counts() (int, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lookups, append([]string(nil), c.dials...)
}

// TestIPLiteralHostIsRefusedBeforeAnyLookupOrDial sends a request to the loopback server by its IP address, with the
// catalog setting off and on.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A host that is an IP address is refused before any lookup or dial, with or without the catalog setting
func TestIPLiteralHostIsRefusedBeforeAnyLookupOrDial(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"First document"}`))
	})
	for _, allow := range []bool{false, true} {
		counter := &loopCountingNetwork{base: lb.network("127.0.0.1")}
		client := &http.Client{Transport: engine.GuardedTransport(counter.network(), allow, lb.roots)}
		_, err := client.Get("https://" + lb.port + "/v1/docs/doc-1")
		if !errors.Is(err, engine.ErrNetworkRefused) {
			t.Fatalf("allow=%v: err = %v, want the network refusal", allow, err)
		}
		lookups, dials := counter.counts()
		if lookups != 0 || len(dials) != 0 {
			t.Fatalf("allow=%v: the IP host reached the resolver (%d) or the dialer (%v)", allow, lookups, dials)
		}
		if n := lb.count(); n != 0 {
			t.Fatalf("allow=%v: the server received %d requests, want none", allow, n)
		}
	}
}

// TestLocalNameIsNeverConnectedTo sends requests to localhost and to a subdomain of it, with the catalog setting off and on.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A local name is never connected to, with or without the catalog setting
func TestLocalNameIsNeverConnectedTo(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"First document"}`))
	})
	port := lb.port[strings.LastIndex(lb.port, ":")+1:]
	for _, allow := range []bool{false, true} {
		for _, host := range []string{"localhost", "api.localhost"} {
			counter := &loopCountingNetwork{base: lb.network("127.0.0.1")}
			client := &http.Client{Transport: engine.GuardedTransport(counter.network(), allow, lb.roots)}
			_, err := client.Get("https://" + host + ":" + port + "/v1/docs/doc-1")
			if !errors.Is(err, engine.ErrNetworkRefused) {
				t.Fatalf("%s allow=%v: err = %v, want the network refusal", host, allow, err)
			}
			if _, dials := counter.counts(); len(dials) != 0 {
				t.Fatalf("%s allow=%v: a connection was made to %v", host, allow, dials)
			}
			if n := lb.count(); n != 0 {
				t.Fatalf("%s allow=%v: the server received %d requests, want none", host, allow, n)
			}
		}
	}
}

// TestCatalogSettingDialsLoopbackAndRefusesTheOthers reads a host that resolves to loopback, unspecified, and multicast
// addresses, with the catalog setting allowing private networks.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: The catalog setting allows a loopback address and still refuses an unspecified or a multicast address
func TestCatalogSettingDialsLoopbackAndRefusesTheOthers(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"First document"}`))
	})
	s := loopSpec(t, "20", loopOps)

	eng, err := engine.New(s, loopSource(t), engine.Ports{
		Credentials: loopCredentials{},
		Clock:       loopClock{},
		Network:     lb.network("127.0.0.1"),
		Roots:       lb.roots,
	}, engine.AllowPrivateNetwork())
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	if _, err := eng.Read(context.Background(), "example-wiki.docs.get", loopDocArgs()); err != nil {
		t.Fatalf("the loopback read fails: %v", err)
	}
	if n := lb.count(); n != 1 {
		t.Fatalf("the loopback server received %d requests, want one", n)
	}

	for _, answer := range []string{"0.0.0.0", "224.0.0.1"} {
		eng, err := engine.New(s, loopSource(t), engine.Ports{
			Credentials: loopCredentials{},
			Clock:       loopClock{},
			Network:     lb.network(answer),
			Roots:       lb.roots,
		}, engine.AllowPrivateNetwork())
		if err != nil {
			t.Fatalf("build the engine for %s: %v", answer, err)
		}
		_, err = eng.Read(context.Background(), "example-wiki.docs.get", loopDocArgs())
		if !errors.Is(err, engine.ErrNetworkRefused) {
			t.Fatalf("%s: err = %v, want the network refusal even with the setting", answer, err)
		}
	}
	if n := lb.count(); n != 1 {
		t.Fatalf("the server received %d requests after the refused addresses, want still one", n)
	}
}

// TestDialOfAnAddressWithoutAPortIsRefusedBeforeTheResolver dials a target that holds a host name and no port, against a
// running server.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: An address without a port is refused before the resolver is asked
func TestDialOfAnAddressWithoutAPortIsRefusedBeforeTheResolver(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"First document"}`))
	})
	counter := &loopCountingNetwork{base: lb.network("127.0.0.1")}
	tr := engine.GuardedTransport(counter.network(), false, lb.roots)
	_, err := tr.DialContext(context.Background(), "tcp", loopHost)
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if lookups, _ := counter.counts(); lookups != 0 {
		t.Fatalf("the resolver was asked %d times for an address without a port", lookups)
	}
}

// TestDroppedConnectionDuringTheBodyHoldsNoKey answers with a body that is cut short by the server closing the connection.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A connection that drops while the reply body is read is reported as unreachable with no key in the text
func TestDroppedConnectionDuringTheBodyHoldsNoKey(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("the response cannot be hijacked")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 1000\r\n\r\n" + `{"title":"Fir`)
		_ = buf.Flush()
	})
	s := loopSpec(t, "20", loopOps)
	_, err := loopEngine(t, s, lb, true, engine.AllowPrivateNetwork()).Read(context.Background(), "example-wiki.docs.get", loopDocArgs())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if strings.Contains(err.Error(), loopKey) {
		t.Fatalf("the error text holds the key: %q", err.Error())
	}
}

// TestLoopbackIsDialedOnlyWithTheCatalogSetting checks the same host through the guard with the setting off and then on.
//
// @scenario specs/libs/retriever-engine/behaviours/network-guard.feature: A loopback address is never dialed without the catalog setting and is dialed with it
func TestLoopbackIsDialedOnlyWithTheCatalogSetting(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"First document"}`))
	})

	off := &loopCountingNetwork{base: lb.network("127.0.0.1")}
	client := &http.Client{Transport: engine.GuardedTransport(off.network(), false, lb.roots)}
	if _, err := client.Get("https://" + loopHost + "/v1/docs/doc-1"); !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("without the setting: err = %v, want the network refusal", err)
	}
	if _, dials := off.counts(); len(dials) != 0 {
		t.Fatalf("a loopback address was dialed without the setting: %v", dials)
	}
	if n := lb.count(); n != 0 {
		t.Fatalf("the server received %d requests without the setting, want none", n)
	}

	on := &loopCountingNetwork{base: lb.network("127.0.0.1")}
	client = &http.Client{Transport: engine.GuardedTransport(on.network(), true, lb.roots)}
	resp, err := client.Get("https://" + loopHost + "/v1/docs/doc-1")
	if err != nil {
		t.Fatalf("with the setting: %v", err)
	}
	_ = resp.Body.Close()
	if _, dials := on.counts(); len(dials) != 1 || dials[0] != "127.0.0.1:443" {
		t.Fatalf("with the setting dialed %v, want the checked address 127.0.0.1:443", dials)
	}
	if n := lb.count(); n != 1 {
		t.Fatalf("the server received %d requests with the setting, want one", n)
	}
}
