//go:build integration

package engine_test

import (
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
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// docHost is the documentation host of the fixture, which the test resolves to the loopback address.
const docHost = "docs.example.test"

// realClock is the clock of the integration tests. It uses the real timer, which is allowed under the integration tag.
type realClock struct{}

// WithTimeout returns a context with a real deadline.
func (realClock) WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}

// Sleep waits on a real timer, or until the context ends.
func (realClock) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// loopback is a real HTTPS server on the loopback address, with a certificate for the documentation host.
type loopback struct {
	srv   *httptest.Server
	roots *x509.CertPool
	hits  atomic.Int32
}

// testCertificate makes a self-signed certificate for the documentation host and the loopback address.
func testCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: docHost},
		DNSNames:     []string{docHost},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
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

// startLoopback starts an HTTPS server on 127.0.0.1 that counts each request and then calls the handler.
func startLoopback(t *testing.T, h http.HandlerFunc) *loopback {
	t.Helper()
	cert, roots := testCertificate(t)
	lb := &loopback{roots: roots}
	lb.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lb.hits.Add(1)
		h(w, r)
	}))
	lb.srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	lb.srv.StartTLS()
	t.Cleanup(lb.srv.Close)
	return lb
}

// realNetwork returns the network of a test: the docs host resolves to the loopback address, and a dial to the checked
// loopback address on the standard port reaches the test server.
func (lb *loopback) realNetwork() engine.Network {
	port := lb.srv.Listener.Addr().String()
	return engine.Network{
		Lookup: func(_ context.Context, host string) ([]netip.Addr, error) {
			if host != docHost {
				return nil, errors.New("the test resolver knows only the documentation host")
			}
			return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
		},
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != "127.0.0.1:443" {
				return nil, errors.New("the test dialer reaches only the checked loopback address")
			}
			var d net.Dialer
			return d.DialContext(ctx, network, port)
		},
	}
}

// realEngine builds an engine for a spec over the real dialer and the loopback server.
func realEngine(t *testing.T, s *spec.Spec, lb *loopback, opts ...engine.Option) *engine.Engine {
	t.Helper()
	eng, err := engine.New(s, testSource(t), engine.Ports{
		Credentials: &fakeCredentials{},
		Clock:       realClock{},
		Network:     lb.realNetwork(),
		Roots:       lb.roots,
	}, opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return eng
}

// TestS5_TheRealDialerRefusesLoopbackWithoutTheCatalogSetting checks the dial control on a real connection. A host that
// resolves to the loopback address is refused, and the server receives no request.
func TestS5_TheRealDialerRefusesLoopbackWithoutTheCatalogSetting(t *testing.T) {
	lb := startLoopback(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"items":[],"next_cursor":null}`))
	})
	eng := realEngine(t, loadSpec(t), lb)
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want ErrNetworkRefused", err)
	}
	if n := lb.hits.Load(); n != 0 {
		t.Fatalf("the refused loopback host received %d requests", n)
	}
}

// TestS5_TheCatalogSettingConnectsOverRealTLS checks that the same run connects when the catalog allows private
// networks, and that the key arrives in the authorization header over TLS.
func TestS5_TheCatalogSettingConnectsOverRealTLS(t *testing.T) {
	var auth string
	lb := startLoopback(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Write([]byte(`{"items":[{"id":"doc-1","title":"First document"}],"next_cursor":null}`))
	})
	eng := realEngine(t, loadSpec(t), lb, engine.AllowPrivateNetwork())
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read over TLS: %v", err)
	}
	if len(rec.Items()) != 1 || rec.Items()[0].Str("id") != "doc-1" {
		t.Fatalf("the mapped reply is wrong: %+v", rec.Items())
	}
	if lb.hits.Load() != 1 || auth != "Bearer "+fixtureKey {
		t.Fatalf("hits = %d, authorization = %q; want one request with the bearer key", lb.hits.Load(), auth)
	}
}

// TestS4_ARealRedirectIsNotFollowed checks that a real 302 reply is refused, and that the server gets one request.
func TestS4_ARealRedirectIsNotFollowed(t *testing.T) {
	lb := startLoopback(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://elsewhere.example.test/v1/docs")
		w.WriteHeader(http.StatusFound)
	})
	eng := realEngine(t, loadSpec(t), lb, engine.AllowPrivateNetwork())
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("err = %v, want ErrRedirected", err)
	}
	if n := lb.hits.Load(); n != 1 {
		t.Fatalf("a redirect caused %d requests, want 1", n)
	}
}

// TestS3_ARealStallHitsTheSpecTimeLimit checks a real stall. The spec lowers the limit to one second, and the server
// never answers until the test ends, so the read exits with the timeout status and sends no second request.
func TestS3_ARealStallHitsTheSpecTimeLimit(t *testing.T) {
	release := make(chan struct{})
	lb := startLoopback(t, func(w http.ResponseWriter, _ *http.Request) {
		<-release
	})
	t.Cleanup(func() { close(release) })
	s := loadSpecFrom(t, replaceIn(t, wikiYAML, "timeout_seconds: 20", "timeout_seconds: 1"))
	eng := realEngine(t, s, lb, engine.AllowPrivateNetwork())
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("err = %v, want ErrTimedOut", err)
	}
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("exit status = %d, want 124", got)
	}
	if n := lb.hits.Load(); n != 1 {
		t.Fatalf("a stalled read caused %d requests, want 1", n)
	}
}
