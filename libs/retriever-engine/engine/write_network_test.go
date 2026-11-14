package engine_test

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// pipeNetwork is the network of the tests that reach the guarded transport. Its lookup answers one public documentation
// address, and its dialer hands the engine one end of an in-memory pipe. The other end is run by behave, so a test decides
// what the source does: refuse the handshake, read the request, or cut the connection. No socket is opened.
type pipeNetwork struct {
	cert    tls.Certificate
	roots   *x509.CertPool
	dials   atomic.Int32
	hits    atomic.Int32
	dialErr error
	behave  func(conn *tls.Conn, hits *atomic.Int32)
}

// newPipeNetwork builds a pipe network whose source presents a certificate for the documentation host.
func newPipeNetwork(t *testing.T, behave func(*tls.Conn, *atomic.Int32)) *pipeNetwork {
	t.Helper()
	cert, roots := pipeCertificate(t)
	return &pipeNetwork{cert: cert, roots: roots, behave: behave}
}

// network returns the engine's view of the pipe network.
func (p *pipeNetwork) network() engine.Network {
	return engine.Network{
		Lookup: func(_ context.Context, _ string) ([]netip.Addr, error) {
			return []netip.Addr{netip.AddrFrom4([4]byte{198, 51, 100, 9})}, nil
		},
		Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
			p.dials.Add(1)
			if p.dialErr != nil {
				return nil, p.dialErr
			}
			client, server := net.Pipe()
			go func() {
				defer server.Close()
				tc := tls.Server(server, &tls.Config{Certificates: []tls.Certificate{p.cert}, MinVersion: tls.VersionTLS12})
				p.behave(tc, &p.hits)
			}()
			return client, nil
		},
	}
}

// pipeCertificate makes a self-signed certificate for the documentation host. Its validity is fixed, so the test reads no
// clock.
func pipeCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "docs.example.test"},
		DNSNames:     []string{"docs.example.test"},
		NotBefore:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool
}

// guardedWriteEngine builds an engine that uses the guarded transport over a pipe network.
func guardedWriteEngine(t *testing.T, n *pipeNetwork) *engine.Engine {
	t.Helper()
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: &fakeCredentials{},
		Clock:       &fakeClock{},
		Network:     n.network(),
		Roots:       n.roots,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return eng
}

// TestS5_AWriteToAPrivateAddressIsRefusedBeforeAnyDial checks that a source name that resolves to a private address is
// refused before the dialer runs, so the write never reaches the address and its outcome is known.
func TestS5_AWriteToAPrivateAddressIsRefusedBeforeAnyDial(t *testing.T) {
	var dials atomic.Int32
	private := engine.Network{
		Lookup: func(context.Context, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.AddrFrom4([4]byte{10, 0, 0, 5})}, nil
		},
		Dial: func(context.Context, string, string) (net.Conn, error) {
			dials.Add(1)
			return nil, errors.New("the dialer must not run")
		},
	}
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: &fakeCredentials{}, Clock: &fakeClock{}, Network: private,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want ErrNetworkRefused", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a refused address reported an unknown outcome, but no request was sent")
	}
	if got := engine.Code("example-wiki", err); got != "retriever.connector.network_refused" {
		t.Fatalf("code = %q, want retriever.connector.network_refused", got)
	}
	if dials.Load() != 0 {
		t.Fatalf("the refused address reached the dialer %d times", dials.Load())
	}
}

// TestAWriteWhoseDialFailsIsKnownAndSendsNothing checks that a connection that never opens is a known failure. The
// write did not leave, so the outcome is not unknown.
func TestAWriteWhoseDialFailsIsKnownAndSendsNothing(t *testing.T) {
	n := newPipeNetwork(t, func(*tls.Conn, *atomic.Int32) {})
	n.dialErr = errors.New("the test dialer refuses the connection")
	eng := guardedWriteEngine(t, n)
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a dial that failed reported an unknown outcome, but no request was sent")
	}
	if n.hits.Load() != 0 {
		t.Fatalf("the source received %d requests after a failed dial", n.hits.Load())
	}
}

// TestAWriteWhoseHandshakeFailsIsKnownAndSendsNothing checks a TLS handshake that fails before the request leaves. The
// source closes the connection at once, so the write's outcome is known, as the Go connectors decide it. Its error is the
// unreachable error, not the unknown outcome.
func TestAWriteWhoseHandshakeFailsIsKnownAndSendsNothing(t *testing.T) {
	n := newPipeNetwork(t, func(*tls.Conn, *atomic.Int32) {})
	eng := guardedWriteEngine(t, n)
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a handshake that failed reported an unknown outcome, but no request was sent")
	}
	if n.dials.Load() != 1 || n.hits.Load() != 0 {
		t.Fatalf("dials = %d, requests = %d; want one dial and no request", n.dials.Load(), n.hits.Load())
	}
}

// TestAWriteCutAfterTheRequestLeftIsAnUnknownOutcomeOverTheGuardedTransport checks a connection that is made, carries the
// request, and then breaks before a reply. The request reached the source, so the outcome is unknown and the write is
// sent once.
func TestAWriteCutAfterTheRequestLeftIsAnUnknownOutcomeOverTheGuardedTransport(t *testing.T) {
	n := newPipeNetwork(t, func(conn *tls.Conn, hits *atomic.Int32) {
		if err := conn.Handshake(); err != nil {
			return
		}
		if _, err := http.ReadRequest(bufio.NewReader(conn)); err == nil {
			hits.Add(1)
		}
	})
	eng := guardedWriteEngine(t, n)
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if n.hits.Load() != 1 || n.dials.Load() != 1 {
		t.Fatalf("dials = %d, requests received = %d; want one connection and one request", n.dials.Load(), n.hits.Load())
	}
}
