//go:build integration

package integration_test

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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The support of the integration bindings. Every name carries the reintg prefix, so the bindings of the other features of
// this package never collide with it. The bindings reach a real loopback server over real TLS, with real timers, and no
// request leaves the machine.

// reintgDocHost is the documentation host of the example spec, which the test resolves to the loopback address.
const reintgDocHost = "docs.example.test"

// reintgHeader is the header of every spec of these bindings. The time limit is the spec's own, and the bindings lower it.
const reintgHeader = `version: 1
kind: example-wiki
summary: Example wiki reads and writes for the integration bindings.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
operations:
`

// reintgSpec validates the default header with the given operations, and fails the test when the document is refused.
func reintgSpec(t *testing.T, operations string) *spec.Spec {
	t.Helper()
	return reintgSpecFrom(t, reintgHeader, operations)
}

// reintgSpecFrom validates the given header with the given operations, and fails the test when the document is refused.
func reintgSpecFrom(t *testing.T, header, operations string) *spec.Spec {
	t.Helper()
	res := spec.Validate([]byte(header + operations))
	if !res.Valid() {
		t.Fatalf("the bound spec is refused: %v", res.Err())
	}
	return res.Spec
}

// reintgSource returns a source of the example kind, on the documentation host, with the example credential name.
func reintgSource(t *testing.T) domain.Source {
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
	endpoint, err := domain.NewEndpoint("https://" + reintgDocHost)
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

// reintgCredentials is the credentials port of the bindings. It returns a fake key, which is not a real key.
type reintgCredentials struct{}

// APIKey returns the fake key for the reference.
func (reintgCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey(ref.Name(), "fixture-key")
}

// reintgClock is the real clock: its time limits and its waits run on real timers.
type reintgClock struct{}

// WithTimeout returns a context with a real deadline.
func (reintgClock) WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}

// Sleep waits on a real timer, or until the context ends.
func (reintgClock) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// reintgLoopback is a real HTTPS server on the loopback address, with a certificate for the documentation host. It counts
// each request it receives.
type reintgLoopback struct {
	srv   *httptest.Server
	roots *x509.CertPool
	hits  atomic.Int32
}

// reintgCertificate makes a self-signed certificate for the documentation host and the loopback address.
func reintgCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: reintgDocHost},
		DNSNames:     []string{reintgDocHost},
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

// reintgStart starts an HTTPS server on 127.0.0.1 that counts each request and then calls the handler.
func reintgStart(t *testing.T, h http.HandlerFunc) *reintgLoopback {
	t.Helper()
	cert, roots := reintgCertificate(t)
	lb := &reintgLoopback{roots: roots}
	lb.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lb.hits.Add(1)
		h(w, r)
	}))
	lb.srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	lb.srv.StartTLS()
	t.Cleanup(lb.srv.Close)
	return lb
}

// reintgNetwork returns the network of a test: the documentation host resolves to the loopback address, and a dial to the
// checked loopback address on the standard port reaches the test server.
func (lb *reintgLoopback) network() engine.Network {
	port := lb.srv.Listener.Addr().String()
	return engine.Network{
		Lookup: func(_ context.Context, host string) ([]netip.Addr, error) {
			if host != reintgDocHost {
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

// reintgEngineOver builds an engine for a spec over the given network, with no injected roots, and with the given options.
func reintgEngineOver(t *testing.T, s *spec.Spec, n engine.Network, opts ...engine.Option) *engine.Engine {
	t.Helper()
	eng, err := engine.New(s, reintgSource(t), engine.Ports{
		Credentials: reintgCredentials{},
		Clock:       reintgClock{},
		Network:     n,
	}, opts...)
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	return eng
}

// reintgEngine builds an engine for a spec over the real dialer and the loopback server, with the given options.
func reintgEngine(t *testing.T, s *spec.Spec, lb *reintgLoopback, opts ...engine.Option) *engine.Engine {
	t.Helper()
	eng, err := engine.New(s, reintgSource(t), engine.Ports{
		Credentials: reintgCredentials{},
		Clock:       reintgClock{},
		Network:     lb.network(),
		Roots:       lb.roots,
	}, opts...)
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	return eng
}

// reintgObjectOp is the single-document read of the example wiki, with one canned example.
const reintgObjectOp = `  - id: example-wiki.docs.one
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
`

// reintgWriteOp is the create write of the example wiki, with one canned example.
const reintgWriteOp = `  - id: example-wiki.docs.create
    summary: Create one document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
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
          title: First
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`

// reintgTimeoutHeader returns the header with the time limit lowered to one second.
func reintgTimeoutHeader() string {
	return strings.Replace(reintgHeader, "timeout_seconds: 20", "timeout_seconds: 1", 1)
}
