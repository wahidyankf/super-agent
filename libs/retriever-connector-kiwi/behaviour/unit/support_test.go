package unit_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/kiwistub"
)

// The support of the unit bindings. Every double is in memory: no clock, file, environment variable, socket, or process
// is read. A gateway is built with NewWithBase, so its connection is a round tripper of this package or of the Kiwi stub.

const (
	// testHost is the endpoint host of the source. It is the Host header of every request and the TLS name.
	testHost = "kiwi.example.test"
	// testBase is the connection base of the in-memory transport. Its name never resolves, and no socket is opened.
	testBase = "https://kiwi-stub.invalid"
	// sentinelKey is a fake key. The stub accepts it, and no error text may contain it.
	sentinelKey = "sentinel-kiwi-key-7d3a91"
	// keyVariable names the key that the credentials double reports.
	keyVariable = "EXAMPLE_KIWI_API_KEY"
)

// epoch is the fixed start of the fake clock.
var epoch = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

// errHandshake is the failure of a TLS handshake that the server's certificate does not pass.
var errHandshake = errors.New("x509: certificate signed by unknown authority")

// must stops the binding when a fixture cannot be built.
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

// missingCreds reports an unset key variable, as the environment adapter does.
type missingCreds struct{}

func (missingCreds) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey(keyVariable, "")
}

// fakeClock is a clock the binding moves. A wait moves it past the wait and is recorded, so pacing and Retry-After are
// asserted without any real waiting.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: epoch}
}

// Now returns the clock's time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep records the wait and moves the clock past it.
func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.sleeps = append(c.sleeps, d)
	c.mu.Unlock()
	return ctx.Err()
}

// Advance moves the clock forward without a wait.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// Sleeps returns every recorded wait, in order.
func (c *fakeClock) Sleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.sleeps...)
}

// gateway returns a gateway for the source whose connection is rt.
func gateway(t *testing.T, rt http.RoundTripper, creds ports.Credentials, opts ...kiwi.Option) *kiwi.Gateway {
	t.Helper()
	gw, err := kiwi.NewWithBase(kiwiSource(t), testBase, &http.Client{Transport: rt}, creds, opts...)
	must(t, err)
	return gw
}

// hermeticGateway returns a gateway whose pacing delay is zero unless opts set one, and whose stamps and waits go to the
// fake clock. No real time passes through it.
func hermeticGateway(t *testing.T, rt http.RoundTripper, creds ports.Credentials, clock *fakeClock, opts ...kiwi.Option) *kiwi.Gateway {
	t.Helper()
	all := append([]kiwi.Option{kiwi.WithMinDelay(0), kiwi.WithClock(clock.Now), kiwi.WithSleep(clock.Sleep)}, opts...)
	return gateway(t, rt, creds, all...)
}

// rig is a gateway over an in-process Kiwi stub, with a fake clock.
type rig struct {
	stub  *kiwistub.Stub
	clock *fakeClock
	gw    *kiwi.Gateway
}

// newRig builds a gateway over the dataset, answered in process by the stub.
func newRig(t *testing.T, d kiwistub.Dataset) rig {
	t.Helper()
	clock := newFakeClock()
	stub := kiwistub.New(sentinelKey, kiwistub.WithDataset(d), kiwistub.WithClock(clock.Now))
	return rig{stub: stub, clock: clock, gw: hermeticGateway(t, stub.RoundTripper(), staticCreds{}, clock)}
}

// sampleDataset is the small Kiwi server most bindings read: one product with two plans, two cases, and the lookup tables.
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

// setOf returns the members of a list as a set.
func setOf(list []string) map[string]bool {
	set := make(map[string]bool, len(list))
	for _, item := range list {
		set[item] = true
	}
	return set
}

// mustContain fails the binding for each wanted text that the text does not hold.
func mustContain(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("the text does not contain %q:\n%s", want, text)
		}
	}
}

// mustNotContain fails the binding for each unwanted text that the text holds.
func mustNotContain(t *testing.T, text string, unwanted ...string) {
	t.Helper()
	for _, bad := range unwanted {
		if strings.Contains(text, bad) {
			t.Errorf("the text holds %q:\n%s", bad, text)
		}
	}
}

// inOrder fails the binding unless each part appears in the text, after the one before it.
func inOrder(t *testing.T, text string, parts ...string) {
	t.Helper()
	last := -1
	for _, part := range parts {
		at := strings.Index(text, part)
		if at < 0 {
			t.Errorf("the text does not contain %q:\n%s", part, text)
			return
		}
		if at < last {
			t.Errorf("%q comes before the part that should precede it:\n%s", part, text)
			return
		}
		last = at
	}
}

// call is one request as an in-memory server receives it: the JSON-RPC method and id, and the headers the adapter sets.
type call struct {
	method string
	id     int64
	host   string
	auth   string
	cookie string
}

// readCall reads the request's method and id, and puts its body back, so the request can still be answered.
func readCall(req *http.Request) call {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	var env struct {
		Method string `json:"method"`
		ID     int64  `json:"id"`
	}
	_ = json.Unmarshal(body, &env)
	return call{method: env.Method, id: env.ID, host: req.Host, auth: req.Header.Get("Authorization"), cookie: req.Header.Get("Cookie")}
}

// httpResponse builds an in-memory reply with a status, optional extra headers, and a body.
func httpResponse(req *http.Request, status int, header http.Header, body string) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")
	return &http.Response{
		Status:        http.StatusText(status),
		StatusCode:    status,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

// rpcReply answers a request with a JSON-RPC 2.0 envelope that holds the request's own id and the given result.
func rpcReply(req *http.Request, c call, result string, header http.Header) *http.Response {
	return httpResponse(req, http.StatusOK, header, fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"result":%s}`, c.id, result))
}

// scripted answers each request with serve. It records every request before serve answers, so a request that serve
// refuses still counts as sent.
type scripted struct {
	mu    sync.Mutex
	seen  []call
	serve func(req *http.Request, c call) (*http.Response, error)
}

// RoundTrip records the request and answers it.
func (s *scripted) RoundTrip(req *http.Request) (*http.Response, error) {
	c := readCall(req)
	s.mu.Lock()
	s.seen = append(s.seen, c)
	s.mu.Unlock()
	return s.serve(req, c)
}

// requests returns every request the scripted server recorded, in order.
func (s *scripted) requests() []call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]call(nil), s.seen...)
}

// shaped answers the methods it names with the given result, and passes every other call to next. It shapes a reply
// that the stub cannot produce, such as a list where an object is expected.
type shaped struct {
	next    http.RoundTripper
	results map[string]string
}

// RoundTrip answers a named method, or passes the request on.
func (s shaped) RoundTrip(req *http.Request) (*http.Response, error) {
	c := readCall(req)
	result, ok := s.results[c.method]
	if !ok {
		return s.next.RoundTrip(req)
	}
	return rpcReply(req, c, result, nil), nil
}

// brokenBody is a reply body that ends with a read error, as a connection that breaks mid-reply does.
type brokenBody struct{ sent bool }

// Read returns a little of the reply, then fails.
func (b *brokenBody) Read(p []byte) (int, error) {
	if !b.sent {
		b.sent = true
		return copy(p, `{"jsonrpc":"2.0"`), nil
	}
	return 0, io.ErrUnexpectedEOF
}

// Close does nothing: the body holds no connection.
func (b *brokenBody) Close() error { return nil }

// handshakeRT fails every connection at its TLS handshake, as a server with an untrusted certificate does. It reports the
// failure to the request's trace, as the transport does, and the request never reaches the server it would dial.
type handshakeRT struct{}

// RoundTrip reports the failed handshake and returns the failure.
func (handshakeRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if tr := httptrace.ContextClientTrace(req.Context()); tr != nil && tr.TLSHandshakeDone != nil {
		tr.TLSHandshakeDone(tls.ConnectionState{}, errHandshake)
	}
	return nil, errHandshake
}

// stepReader is a fixed entropy source, so a stub built by a binding makes the same certificate each run.
type stepReader struct{ state byte }

// Read fills p with a fixed sequence.
func (r *stepReader) Read(p []byte) (int, error) {
	for i := range p {
		r.state = r.state*31 + 17
		p[i] = r.state
	}
	return len(p), nil
}

// parkedListener is a listening socket that never accepts, so a stub can start without a real port.
type parkedListener struct {
	done chan struct{}
	once sync.Once
}

// newParkedListener returns a listener that accepts nothing until it is closed.
func newParkedListener() *parkedListener {
	return &parkedListener{done: make(chan struct{})}
}

// Accept blocks until the listener is closed.
func (l *parkedListener) Accept() (net.Conn, error) {
	<-l.done
	return nil, net.ErrClosed
}

// Close releases the blocked Accept.
func (l *parkedListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

// Addr returns a loopback address that no socket uses.
func (l *parkedListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9}
}

// caPEM returns the certificate of a stub that starts on a parked listener. The key and the signature come from a fixed
// source, so the certificate is the same on every run.
func caPEM(t *testing.T) []byte {
	t.Helper()
	fixed := func() time.Time { return epoch }
	stub := kiwistub.New(sentinelKey,
		kiwistub.WithClock(fixed),
		kiwistub.WithRandom(&stepReader{state: 1}),
		kiwistub.WithListener(func(string, string) (net.Listener, error) { return newParkedListener(), nil }))
	must(t, stub.Start())
	t.Cleanup(stub.Close)
	return stub.CAPEM()
}
