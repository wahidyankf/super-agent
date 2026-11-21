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
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// loopHost is the documentation host of the bindings. The test resolver maps it to the loopback address.
const loopHost = "docs.example.test"

// loopKey is a fake key for the local test server. It is not a real credential.
const loopKey = "integration-fake-key"

// loopHeader is the header of every spec of the integration bindings. Its time limit is replaced for the stall tests.
const loopHeader = `version: 1
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

// loopOps are one read and one write of a document, on the example host.
const loopOps = `  - id: example-wiki.docs.get
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

// loopSpec returns the validated spec of the bindings, with the time limit set to timeoutSeconds.
func loopSpec(t *testing.T, timeoutSeconds string, ops string) *spec.Spec {
	t.Helper()
	doc := strings.Replace(loopHeader, "timeout_seconds: 20", "timeout_seconds: "+timeoutSeconds, 1) + ops
	res := spec.Validate([]byte(doc))
	if !res.Valid() {
		t.Fatalf("the bound spec is refused: %v", res.Err())
	}
	return res.Spec
}

// loopSource returns a source of the example kind on the documentation host.
func loopSource(t *testing.T) domain.Source {
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
	endpoint, err := domain.NewEndpoint("https://" + loopHost)
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

// loopCredentials is the credentials port of the bindings. It returns the fake key.
type loopCredentials struct{}

// APIKey returns the fake key under the name it was asked for.
func (loopCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey(ref.Name(), loopKey)
}

// loopClock is the clock of the bindings. It uses real timers, which the integration tag allows.
type loopClock struct{}

// WithTimeout returns a context with a real deadline.
func (loopClock) WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}

// Sleep waits on a real timer, or until the context ends.
func (loopClock) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// loopRequest is one request that the local server received.
type loopRequest struct {
	method string
	auth   string
	tls    bool
	body   string
}

// loopServer is a real HTTPS server on the loopback address. It records each request it receives, and then calls its
// handler. The certificate names the documentation host, so the client verifies it against the roots of the server.
type loopServer struct {
	srv   *httptest.Server
	roots *x509.CertPool
	port  string

	mu   sync.Mutex
	reqs []loopRequest
}

// count returns how many requests the server received.
func (lb *loopServer) count() int {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return len(lb.reqs)
}

// received returns a copy of the requests the server received.
func (lb *loopServer) received() []loopRequest {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return append([]loopRequest(nil), lb.reqs...)
}

// loopCertificate makes a self-signed certificate for the documentation host and the loopback address.
func loopCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: loopHost},
		DNSNames:     []string{loopHost},
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

// startLoopServer starts the HTTPS server on 127.0.0.1 with the handler. The server stops when the test ends.
func startLoopServer(t *testing.T, h http.HandlerFunc) *loopServer {
	t.Helper()
	cert, roots := loopCertificate(t)
	lb := &loopServer{roots: roots}
	lb.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		lb.mu.Lock()
		lb.reqs = append(lb.reqs, loopRequest{method: r.Method, auth: r.Header.Get("Authorization"), tls: r.TLS != nil, body: string(body)})
		lb.mu.Unlock()
		h(w, r)
	}))
	lb.srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	lb.srv.StartTLS()
	t.Cleanup(lb.srv.Close)
	lb.port = lb.srv.Listener.Addr().String()
	return lb
}

// loopNetwork returns the network port of a binding. The documentation host resolves to answer, and a dial to the checked
// loopback address on the standard port reaches the test server; any other dial is refused by the test dialer.
func (lb *loopServer) network(answer string) engine.Network {
	return engine.Network{
		Lookup: func(_ context.Context, host string) ([]netip.Addr, error) {
			if host != loopHost {
				return nil, errors.New("the test resolver knows only the documentation host")
			}
			return []netip.Addr{netip.MustParseAddr(answer)}, nil
		},
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != "127.0.0.1:443" {
				return nil, errors.New("the test dialer reaches only the checked loopback address")
			}
			var d net.Dialer
			return d.DialContext(ctx, network, lb.port)
		},
	}
}

// loopEngine builds an engine for a spec over the real dialer and the server. The roots are the server's when trusted is
// true, and the system roots otherwise.
func loopEngine(t *testing.T, s *spec.Spec, lb *loopServer, trusted bool, opts ...engine.Option) *engine.Engine {
	t.Helper()
	ports := engine.Ports{
		Credentials: loopCredentials{},
		Clock:       loopClock{},
		Network:     lb.network("127.0.0.1"),
	}
	if trusted {
		ports.Roots = lb.roots
	}
	eng, err := engine.New(s, loopSource(t), ports, opts...)
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	return eng
}

// loopDocArgs are the arguments of the read of doc-1.
func loopDocArgs() domain.Args {
	return domain.NewArgs(map[string]any{"id": "doc-1"})
}

// loopTitleArgs are the arguments of the write of one document.
func loopTitleArgs() domain.Args {
	return domain.NewArgs(map[string]any{"title": "First document"})
}

// TestReadToLoopbackIsRefusedWithoutTheCatalogSetting reads a host that resolves to the loopback address, with the setting
// off, against a server that would answer.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A read to a loopback test server is refused without the catalog setting and the server receives no request
func TestReadToLoopbackIsRefusedWithoutTheCatalogSetting(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"First document"}`))
	})
	s := loopSpec(t, "20", loopOps)
	_, err := loopEngine(t, s, lb, true).Read(context.Background(), "example-wiki.docs.get", loopDocArgs())
	if !errors.Is(err, engine.ErrNetworkRefused) {
		t.Fatalf("err = %v, want the network refusal", err)
	}
	if n := lb.count(); n != 0 {
		t.Fatalf("the server received %d requests, want none", n)
	}
}

// TestReadToLoopbackConnectsOverTLSWithTheCatalogSetting reads the loopback server with the catalog setting allowing private
// networks.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A read to a loopback test server connects over TLS with the catalog setting and sends the key in the authorization header
func TestReadToLoopbackConnectsOverTLSWithTheCatalogSetting(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"First document"}`))
	})
	s := loopSpec(t, "20", loopOps)
	rec, err := loopEngine(t, s, lb, true, engine.AllowPrivateNetwork()).Read(context.Background(), "example-wiki.docs.get", loopDocArgs())
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := rec.Str("title"); got != "First document" {
		t.Fatalf("title = %q, want the reply", got)
	}
	reqs := lb.received()
	if len(reqs) != 1 {
		t.Fatalf("the server received %d requests, want one", len(reqs))
	}
	if !reqs[0].tls {
		t.Fatal("the request did not arrive over TLS")
	}
	if want := "Bearer " + loopKey; reqs[0].auth != want {
		t.Fatalf("authorization header = %q, want the key in the bearer form", reqs[0].auth)
	}
}

// TestRealRedirectIsRefusedAndNotFollowed answers the read with a 302 redirect to another path.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A real 302 reply to a read is refused and the server receives one request
func TestRealRedirectIsRefusedAndNotFollowed(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+loopHost+"/v1/elsewhere", http.StatusFound)
	})
	s := loopSpec(t, "20", loopOps)
	_, err := loopEngine(t, s, lb, true, engine.AllowPrivateNetwork()).Read(context.Background(), "example-wiki.docs.get", loopDocArgs())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("err = %v, want ErrRedirected", err)
	}
	if n := lb.count(); n != 1 {
		t.Fatalf("the server received %d requests, want exactly one", n)
	}
}

// TestRealStallHitsTheTimeLimitWithOneRequest reads from a server that never answers, under a one-second time limit.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A real stall hits the spec time limit and exits with the timeout status without a second request
func TestRealStallHitsTheTimeLimitWithOneRequest(t *testing.T) {
	lb := startLoopServer(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	s := loopSpec(t, "1", loopOps)
	_, err := loopEngine(t, s, lb, true, engine.AllowPrivateNetwork()).Read(context.Background(), "example-wiki.docs.get", loopDocArgs())
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("exit status = %d (err %v), want the timeout status %d", got, err, engine.ExitTimedOut)
	}
	if n := lb.count(); n != 1 {
		t.Fatalf("the server received %d requests, want exactly one", n)
	}
}

// TestWriteTimeoutAfterTheRequestLeftIsAnUnknownOutcome writes to a server that reads nothing and answers nothing until the
// test ends, under a one-second time limit.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A write whose connection times out after the request reached a local server is an unknown outcome and is sent once
func TestWriteTimeoutAfterTheRequestLeftIsAnUnknownOutcome(t *testing.T) {
	release := make(chan struct{})
	lb := startLoopServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	})
	t.Cleanup(func() { close(release) })
	s := loopSpec(t, "1", loopOps)
	_, err := loopEngine(t, s, lb, true, engine.AllowPrivateNetwork()).Write(context.Background(), "example-wiki.docs.create", loopTitleArgs())
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if n := lb.count(); n != 1 {
		t.Fatalf("the server received %d requests, want exactly one", n)
	}
}

// TestWriteOverTLSIsSentOnceAndMapped writes to a server that accepts the document.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A write over TLS to a local server is sent once, carries the key in the authorization header, and maps its declared field
func TestWriteOverTLSIsSentOnceAndMapped(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"doc-1"}`))
	})
	s := loopSpec(t, "20", loopOps)
	rec, err := loopEngine(t, s, lb, true, engine.AllowPrivateNetwork()).Write(context.Background(), "example-wiki.docs.create", loopTitleArgs())
	if err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	reqs := lb.received()
	if len(reqs) != 1 {
		t.Fatalf("the server received %d requests, want exactly one", len(reqs))
	}
	if want := "Bearer " + loopKey; reqs[0].auth != want {
		t.Fatalf("authorization header = %q, want the key in the bearer form", reqs[0].auth)
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want the declared field", got)
	}
}

// TestWriteHandshakeFailureIsUnreachableAndNotSent writes to a server whose certificate the client does not trust.
//
// @scenario specs/libs/retriever-engine/behaviours/loopback-connections.feature: A TLS handshake that fails against a local server is a known failure and the write is not sent
func TestWriteHandshakeFailureIsUnreachableAndNotSent(t *testing.T) {
	lb := startLoopServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	s := loopSpec(t, "20", loopOps)
	_, err := loopEngine(t, s, lb, false, engine.AllowPrivateNetwork()).Write(context.Background(), "example-wiki.docs.create", loopTitleArgs())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if n := lb.count(); n != 0 {
		t.Fatalf("the server received %d requests, want none", n)
	}
}
