//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// The support of the integration bindings. Each binding serves the testkit's Kiwi stub on a real TLS listener of the loopback
// address. The certificate is valid for the endpoint host and for 127.0.0.1, so the adapter's TLS name check runs against a
// real handshake. No binding reaches a host outside the machine, and no binding needs a real secret.

const (
	testHost    = "kiwi.example.test"
	sentinelKey = "sentinel-kiwi-key-7d3a91"
	keyVariable = "EXAMPLE_KIWI_API_KEY"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
}

// kiwiSource returns the Kiwi source of the bindings, whose endpoint is the test host.
func kiwiSource(t *testing.T) domain.Source {
	t.Helper()
	id, err := domain.NewSourceID("kiwi")
	must(t, err)
	endpoint, err := domain.NewEndpoint("https://" + testHost)
	must(t, err)
	ref, err := domain.NewCredentialRef(keyVariable)
	must(t, err)
	tags, err := domain.NewTagSet()
	must(t, err)
	return domain.NewSource(id, domain.KindKiwi, tags, endpoint, ref)
}

// staticCreds returns the fake key for every request.
type staticCreds struct{}

func (staticCreds) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey(keyVariable, sentinelKey)
}

// missingCreds reports an unset key variable.
type missingCreds struct{}

func (missingCreds) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey(keyVariable, "")
}

// recorder is a wait that records each pause and returns at once, so a Retry-After is seen without a real wait.
type recorder struct {
	mu    sync.Mutex
	waits []time.Duration
}

// Sleep records the wait.
func (r *recorder) Sleep(ctx context.Context, d time.Duration) error {
	r.mu.Lock()
	r.waits = append(r.waits, d)
	r.mu.Unlock()
	return ctx.Err()
}

// Waits returns every recorded wait, in order.
func (r *recorder) Waits() []time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Duration(nil), r.waits...)
}

// sampleDataset is the small Kiwi server most bindings read: two products, two plans, two cases, and the lookup tables.
func sampleDataset() kiwistub.Dataset {
	return kiwistub.Dataset{
		Products: []kiwistub.Product{{ID: 1, Name: "Example Portal"}, {ID: 2, Name: "Empty"}},
		Plans: []kiwistub.Plan{
			{ID: 1, Name: "Smoke", Product: 1, Text: "Run before each release."},
			{ID: 2, Name: "Login", Product: 1, Parent: 1, Text: "Check the login flow."},
		},
		Cases: []kiwistub.Case{
			{ID: 12, Summary: "Login works", Text: "Open the login page.", Plans: []int{2}, Sortkey: 1, Status: 1, Priority: 2, Category: 1},
			{ID: 13, Summary: "Logout works", Text: "Log out.", Plans: []int{2}, Sortkey: 2, Status: 1, Priority: 2, Category: 1},
		},
		Categories: []kiwistub.Category{{ID: 1, Name: "Functional", Product: 1}, {ID: 4, Name: "Regression", Product: 1}, {ID: 5, Name: "Regression", Product: 1}},
		Priorities: []kiwistub.Priority{{ID: 2, Value: "P1"}},
		Statuses:   []kiwistub.Status{{ID: 1, Name: "CONFIRMED"}, {ID: 2, Name: "DRAFT"}},
	}
}

// newStub returns a stub that accepts the test key and serves the dataset.
func newStub(d kiwistub.Dataset) *kiwistub.Stub {
	return kiwistub.New(sentinelKey, kiwistub.WithDataset(d))
}

// loopbackCertificate returns a self-signed certificate for the endpoint host and for the loopback address, with its PEM form,
// which a client trusts.
func loopbackCertificate(t *testing.T) (tls.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(t, err)
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: testHost},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{testHost},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	must(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// wireSeen is what the server saw of one request: its TLS name, its Host header, and its Cookie header.
type wireSeen struct {
	ServerName string
	Host       string
	Cookie     string
	Secure     bool
}

// wireLog keeps what the server saw of each request. It is outside every wrapper, so it sees a request that a wrapper answers
// itself.
type wireLog struct {
	mu   sync.Mutex
	rows []wireSeen
}

// add keeps one request.
func (l *wireLog) add(s wireSeen) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rows = append(l.rows, s)
}

// all returns every request kept, in order.
func (l *wireLog) all() []wireSeen {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]wireSeen(nil), l.rows...)
}

// count returns how many requests were kept.
func (l *wireLog) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.rows)
}

// seenOf returns what the server sees of a request.
func seenOf(r *http.Request) wireSeen {
	s := wireSeen{Host: r.Host, Cookie: r.Header.Get("Cookie"), Secure: r.TLS != nil}
	if r.TLS != nil {
		s.ServerName = r.TLS.ServerName
	}
	return s
}

// kiwiServer is a running TLS listener over the stub. Base is its connection base on the loopback address, Client trusts its
// certificate, CA is the PEM a client trusts, and Log keeps what it saw.
type kiwiServer struct {
	Base   string
	Client *http.Client
	CA     []byte
	Stub   *kiwistub.Stub
	Log    *wireLog
}

// serveKiwi serves the stub, or the handler that wrap builds around it, on a loopback TLS listener. The server closes when the
// test ends.
func serveKiwi(t *testing.T, stub *kiwistub.Stub, wrap func(http.Handler) http.Handler) kiwiServer {
	t.Helper()
	cert, caPEM := loopbackCertificate(t)
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("the test certificate did not parse")
	}
	var inner http.Handler = stub
	if wrap != nil {
		inner = wrap(stub)
	}
	log := &wireLog{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.add(seenOf(r))
		inner.ServeHTTP(w, r)
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	must(t, err)
	srv := &http.Server{
		Handler:   handler,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12},
	}
	go func() { _ = srv.Serve(tls.NewListener(ln, srv.TLSConfig)) }()
	t.Cleanup(func() { _ = srv.Close() })
	client := &http.Client{Transport: &http.Transport{
		Proxy:           nil,
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}}
	return kiwiServer{Base: "https://" + ln.Addr().String(), Client: client, CA: caPEM, Stub: stub, Log: log}
}

// rpcMux records the method of each call the server receives. It answers the methods named in results with the JSON-RPC result
// given, and passes every other call on to the stub. The stub cannot give a shaped result, so this wrapper gives one.
type rpcMux struct {
	mu      sync.Mutex
	methods []string
	results map[string]string
}

// wrap is the handler wrapper of the mux.
func (m *rpcMux) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var env struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(body, &env)
		m.mu.Lock()
		m.methods = append(m.methods, env.Method)
		result, shaped := m.results[env.Method]
		m.mu.Unlock()
		if !shaped {
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
			return
		}
		id := string(env.ID)
		if id == "" {
			id = "1"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + id + `,"result":` + result + `}`))
	})
}

// count returns how many calls of the method the server received.
func (m *rpcMux) count(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, got := range m.methods {
		if got == method {
			n++
		}
	}
	return n
}

// breakBody answers 200 with the start of a body that is cut short, then closes the connection. The declared length is longer
// than what is sent, so the client sees the body end early.
func breakBody(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0"`))
		_ = http.NewResponseController(w).Flush()
		if conn, _, err := http.NewResponseController(w).Hijack(); err == nil {
			_ = conn.Close()
		}
	})
}

// gatewayAt returns a gateway that connects to the server, with no pacing delay unless opts set one.
func gatewayAt(t *testing.T, srv kiwiServer, creds ports.Credentials, opts ...kiwi.Option) *kiwi.Gateway {
	t.Helper()
	all := append([]kiwi.Option{kiwi.WithMinDelay(0)}, opts...)
	gw, err := kiwi.NewWithBase(kiwiSource(t), srv.Base, srv.Client, creds, all...)
	must(t, err)
	return gw
}

// untrustingGateway returns a gateway that trusts only the system roots, so the stub's certificate fails its handshake.
func untrustingGateway(t *testing.T, base string) *kiwi.Gateway {
	t.Helper()
	gw, err := kiwi.NewWithBase(kiwiSource(t), base, &http.Client{}, staticCreds{}, kiwi.WithMinDelay(0))
	must(t, err)
	return gw
}

// treeOf reads the collection tree of a product the way a caller holds the text: the domain check comes first.
func treeOf(gw *kiwi.Gateway, raw string) ([]domain.TreeNode, error) {
	id, err := domain.NewCollectionID(raw)
	if err != nil {
		return nil, err
	}
	return gw.CollectionTree(context.Background(), id)
}

// docOf reads the document of a plan or a case the way a caller holds the text: the domain check comes first.
func docOf(gw *kiwi.Gateway, raw string) (domain.SourceDocument, error) {
	id, err := domain.NewDocumentID(raw)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	return gw.Document(context.Background(), id)
}

// callOf runs one call of a kind: a read of the product list, or a write that adds a case to a plan.
func callOf(gw *kiwi.Gateway, kind string) error {
	ctx := context.Background()
	if kind == "read" {
		_, err := gw.ListCollections(ctx)
		return err
	}
	_, err := gw.Write(ctx, domain.OpKiwiPlansAddCase, domain.NewArgs(map[string]any{"planId": "plan-2", "caseId": "case-12"}))
	return err
}

// isKiwiKind reports whether err names the Kiwi kind.
func isKiwiKind(err error) bool {
	var ke domain.KindError
	return errors.As(err, &ke) && ke.Kind == domain.KindKiwi
}

// nodeIDs returns the ID of each node, in order.
func nodeIDs(nodes []domain.TreeNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.ID)
	}
	return out
}
