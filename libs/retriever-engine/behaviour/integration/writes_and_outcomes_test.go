//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// reintgWriteArgs is the argument set of one create of the example wiki.
func reintgWriteArgs() domain.Args {
	return domain.NewArgs(map[string]any{"title": "First"})
}

// TestWritesAndOutcomes_RealRefusalBeforeRequestLeaves binds each failure that happens before a real request leaves to a known
// failure that is not an unknown outcome. A closed port is refused by a real dial, a plain TCP server fails the TLS handshake,
// and a private address is refused by the check before any dial.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A failure before the request leaves is a known failure and sends nothing
func TestWritesAndOutcomes_RealRefusalBeforeRequestLeaves(t *testing.T) {
	t.Run("a connection that never opens", func(t *testing.T) {
		closed, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := closed.Addr().String()
		_ = closed.Close()
		n := engine.Network{
			Lookup: func(context.Context, string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
			},
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		}
		eng := reintgEngineOver(t, reintgSpec(t, reintgWriteOp), n, engine.AllowPrivateNetwork())
		_, err = eng.Write(context.Background(), "example-wiki.docs.create", reintgWriteArgs())
		if err == nil || errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("the error is %v, want a known failure", err)
		}
	})

	t.Run("a TLS handshake that fails", func(t *testing.T) {
		plain, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = plain.Close() })
		go func() {
			for {
				c, err := plain.Accept()
				if err != nil {
					return
				}
				_, _ = c.Write([]byte("HTTP/1.1 201 Created\r\nContent-Length: 0\r\n\r\n"))
				_ = c.Close()
			}
		}()
		addr := plain.Addr().String()
		n := engine.Network{
			Lookup: func(context.Context, string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
			},
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		}
		eng := reintgEngineOver(t, reintgSpec(t, reintgWriteOp), n, engine.AllowPrivateNetwork())
		_, err = eng.Write(context.Background(), "example-wiki.docs.create", reintgWriteArgs())
		if err == nil || errors.Is(err, domain.ErrWriteOutcomeUnknown) {
			t.Fatalf("the error is %v, want a known failure", err)
		}
	})

	t.Run("a source name on a private address", func(t *testing.T) {
		var dials atomic.Int32
		n := engine.Network{
			Lookup: func(context.Context, string) ([]netip.Addr, error) {
				return []netip.Addr{netip.AddrFrom4([4]byte{10, 0, 0, 5})}, nil
			},
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				dials.Add(1)
				var d net.Dialer
				return d.DialContext(ctx, network, address)
			},
		}
		eng := reintgEngineOver(t, reintgSpec(t, reintgWriteOp), n)
		_, err := eng.Write(context.Background(), "example-wiki.docs.create", reintgWriteArgs())
		if !errors.Is(err, engine.ErrNetworkRefused) {
			t.Fatalf("the error is %v, want the refused address", err)
		}
		if d := dials.Load(); d != 0 {
			t.Fatalf("the source was dialed %d times, want none", d)
		}
	})
}

// TestWritesAndOutcomes_RealRedirectWriteRefusedAfterOneRequest binds a write whose source answers with a redirect over real
// TLS to a refusal after exactly one request received by the server.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write whose source answers with a redirect is refused after one request
func TestWritesAndOutcomes_RealRedirectWriteRefusedAfterOneRequest(t *testing.T) {
	lb := reintgStart(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://elsewhere.example.test/v1/docs")
		w.WriteHeader(http.StatusFound)
	})
	eng := reintgEngine(t, reintgSpec(t, reintgWriteOp), lb, engine.AllowPrivateNetwork())
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", reintgWriteArgs())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("the error is %v, want the redirect refusal", err)
	}
	if n := lb.hits.Load(); n != 1 {
		t.Fatalf("the server received %d requests, want exactly one", n)
	}
}
