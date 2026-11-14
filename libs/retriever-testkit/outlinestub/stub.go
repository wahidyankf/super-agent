// Package outlinestub is a scripted stand-in for the Outline API. It replays the
// fixtures in testdata, applies scripted faults, and records every request, so a
// test can assert the calls the client made.
//
// The fixtures are sanitized, hand-written copies of the public response shapes
// in the Outline OpenAPI schema. They hold no key, no private text, and only fake
// content. Re-recording them is a manual step; nothing here calls the real API.
//
// The same Stub serves two transports. New gives an in-process RoundTripper with
// no socket, for unit tests. Start listens on a TLS loopback port through
// httptest, for integration and end-to-end tests.
package outlinestub

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// apiPrefix is the path prefix of every Outline method.
const apiPrefix = "/api/"

// maxRequestBody caps what the stub reads from a request, so a broken test cannot exhaust memory.
const maxRequestBody = 1 << 20

// errConnectionDropped is what the in-process transport returns for a dropped connection.
var errConnectionDropped = errors.New("connection reset by peer")

// errRefusedHost is what the in-process transport returns for a host it does not answer. It names no host.
var errRefusedHost = errors.New("outlinestub: refused a host that is not a loopback address or a .invalid name")

// answerableHost reports whether the in-process stub may answer a request to host: a loopback address, localhost, or a
// name under .invalid, which never resolves on the internet (RFC 2606).
func answerableHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".invalid") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Collection is one collection the stub holds.
type Collection struct {
	ID   string
	Name string
}

// Document is one document the stub holds. ParentID is "" for a root document.
// Documents of one collection are listed in slice order among their siblings.
type Document struct {
	ID           string
	CollectionID string
	ParentID     string
	Title        string
	URL          string
	Markdown     string
}

// Fault is a scripted answer that replaces the normal replay.
//
// Method, when set, limits the fault to requests for that Outline method. Hang and
// Drop act on the transport: Hang never answers, and Drop closes the connection.
// Otherwise Status sets the HTTP status. A zero Status with no Body
// or EchoAuthorization leaves the normal replay in place. A status of 400 or
// above gets the matching sanitized error body unless Body or EchoAuthorization
// replaces it.
type Fault struct {
	Method            string
	Status            int
	RetryAfter        string
	Location          string
	Body              string
	EchoAuthorization bool
	Hang              bool
	Drop              bool
}

// Request is one request the stub received, in arrival order. Method is the Outline
// method name, such as "collections.list". HTTPMethod is the HTTP verb, such as "POST".
// Secure is true when the request arrived over TLS, or, for an in-process request, when
// its URL scheme is https.
type Request struct {
	Method        string
	HTTPMethod    string
	Secure        bool
	Path          string
	Host          string
	Authorization string
	Offset        int
	Limit         int
	ID            string
	At            time.Time
	Status        int
	// ContentType and APIVersion are the Content-Type and X-API-Version headers as sent. ClientVersion is the
	// X-Client-Version header, which Retriever must never send.
	ContentType   string
	APIVersion    string
	ClientVersion string
	// Body is the JSON body as the client wrote it, with every key. A missing body is nil.
	Body map[string]any
}

// Option configures a Stub at construction.
type Option func(*Stub)

// WithCollections sets the collections the stub holds, in listing order.
func WithCollections(collections ...Collection) Option {
	return func(s *Stub) {
		s.collections = append([]Collection(nil), collections...)
	}
}

// WithDocuments sets the documents the stub holds.
func WithDocuments(documents ...Document) Option {
	return func(s *Stub) {
		s.documents = append([]Document(nil), documents...)
	}
}

// WithClock sets the clock that stamps each request. The default is time.Now.
func WithClock(now func() time.Time) Option {
	return func(s *Stub) {
		s.now = now
	}
}

// WithFullPages makes collections.list answer a full page of 25 items whatever
// the offset, so a test can drive the client into its page cap.
func WithFullPages() Option {
	return func(s *Stub) {
		s.fullPages = true
	}
}

// NumberedCollections returns n collections named col-001 to col-nnn, in order.
func NumberedCollections(n int) []Collection {
	out := make([]Collection, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, Collection{
			ID:   fmt.Sprintf("col-%03d", i),
			Name: fmt.Sprintf("Collection %03d", i),
		})
	}
	return out
}

// Account is the signed-in user and team that auth.info answers with. Its e-mail address is fixed, and it is never
// meant to reach an output.
type Account struct {
	User string
	Team string
}

// defaultAccount is the account of a stub that a scenario has not set.
var defaultAccount = Account{User: "Fixture Author", Team: "Fixture Team"}

// Stub is a scripted Outline API. Its methods are safe for concurrent use.
type Stub struct {
	mu          sync.Mutex
	collections []Collection
	documents   []Document
	fullPages   bool
	now         func() time.Time
	fault       Fault
	faultLeft   int // -1 applies the fault to every request; n applies it to the next n
	requests    []Request
	inFlight    int
	peak        int
	done        chan struct{}
	closeOnce   sync.Once
	server      *httptest.Server
	// listen opens the listening socket of Start. Nil means the loopback default.
	listen func(network, address string) (net.Listener, error)
	// certNow dates the certificate of Start. Nil means the package clock, tlsNow.
	certNow func() time.Time

	// unoffered names the methods the server does not offer: each answers 404 "Endpoint not found", as the router does.
	unoffered map[string]bool
	// replies holds the fixed answers that SetReply gives, by method.
	replies map[string]reply
	// ignoreVersion makes documents.info answer the unwrapped shape whatever X-API-Version says.
	ignoreVersion bool
	// editorOnly makes documents.info answer editor data with no Markdown text, the shape of version 3.
	editorOnly bool
	account    Account
	// refused counts the in-process requests whose host was neither a loopback address nor a .invalid name. Such a
	// request is never answered and never recorded, so a test can prove that nothing reached a real host.
	refused int
}

// Refused returns how many in-process requests the stub refused because their host was not a loopback address or a
// .invalid name. A test that sends writes asserts that it is zero.
func (s *Stub) Refused() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refused
}

// New returns a Stub that answers through RoundTripper only. It opens no socket.
func New(opts ...Option) *Stub {
	s := &Stub{
		now:  time.Now,
		done: make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start returns a Stub listening on a TLS loopback port. Tb.Cleanup closes it.
func Start(tb testing.TB, opts ...Option) *Stub {
	tb.Helper()
	s := New(opts...)
	listen := s.listen
	if listen == nil {
		listen = listenTCP
	}
	now := tlsNow
	if s.certNow != nil {
		now = s.certNow
	}
	srv, err := NewTLSServerWith(s, listen, now)
	if err != nil {
		tb.Fatalf("outlinestub: start the TLS server: %v", err)
	}
	s.server = srv
	tb.Cleanup(s.Close)
	return s
}

// WithListener sets the function that opens the listening socket of Start. The default opens a real loopback port. A unit test
// passes an in-memory listener, so the stub opens no socket.
func WithListener(listen func(network, address string) (net.Listener, error)) Option {
	return func(s *Stub) { s.listen = listen }
}

// WithCertificateClock sets the clock that dates the certificate of Start. A client that verifies at the same clock then
// accepts it. Without the option the certificate is dated by the package clock.
func WithCertificateClock(now func() time.Time) Option {
	return func(s *Stub) { s.certNow = now }
}

// URL returns the base URL of a started Stub, or "" for an in-process Stub.
func (s *Stub) URL() string {
	if s.server == nil {
		return ""
	}
	return s.server.URL
}

// CAPEM returns the PEM certificate a client must trust to reach a started Stub.
func (s *Stub) CAPEM() ([]byte, error) {
	if s.server == nil {
		return nil, errors.New("outlinestub: the stub is not started")
	}
	cert := s.server.Certificate()
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), nil
}

// writeFile writes the CA file of WriteCAFile. A unit test replaces it, so it writes no file.
var writeFile = os.WriteFile

// WriteCAFile writes the CA PEM into dir and returns its path.
func (s *Stub) WriteCAFile(dir string) (string, error) {
	data, err := s.CAPEM()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "outline-stub-ca.pem")
	if err := writeFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("outlinestub: write CA file: %w", err)
	}
	return path, nil
}

// Client returns an HTTP client that trusts a started Stub's certificate.
func (s *Stub) Client() *http.Client {
	if s.server == nil {
		return nil
	}
	return s.server.Client()
}

// RoundTripper returns a transport that answers in-process, with no socket.
func (s *Stub) RoundTripper() http.RoundTripper {
	return inProcess{stub: s}
}

// SetFault applies f to every request from now on. A zero Fault clears it.
func (s *Stub) SetFault(f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fault = f
	s.faultLeft = -1
}

// SetFaultFirst applies f to the next n requests only, then the normal replay resumes.
func (s *Stub) SetFaultFirst(n int, f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fault = f
	s.faultLeft = n
}

// SetDataset replaces the collections and documents the stub holds, from the next request on.
// It copies both slices, so the caller may change its own slices afterwards. It is safe for
// concurrent use, and a request already in flight answers from the data it started with.
func (s *Stub) SetDataset(collections []Collection, documents []Document) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collections = append([]Collection(nil), collections...)
	s.documents = append([]Document(nil), documents...)
}

// SetFullPages turns the full-page answer of collections.list on or off, from the next request on.
// It is safe for concurrent use.
func (s *Stub) SetFullPages(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fullPages = on
}

// SetUnoffered makes the stub refuse the named methods as an older or newer router would: a 404 whose message is
// "Endpoint not found". It replaces the earlier list, and it is safe for concurrent use.
func (s *Stub) SetUnoffered(methods ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unoffered = map[string]bool{}
	for _, m := range methods {
		s.unoffered[m] = true
	}
}

// SetReply answers every call of one Outline method with the given status and JSON body, in place of the replay. A
// scenario uses it to state the exact reply it wants a read to project. It replaces an earlier reply of the method, and
// it is safe for concurrent use.
func (s *Stub) SetReply(method string, status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// The map is copied, never changed in place, because a snapshot holds the map it was taken with.
	next := make(map[string]reply, len(s.replies)+1)
	for m, r := range s.replies {
		next[m] = r
	}
	next[method] = reply{
		status: status,
		header: http.Header{"Content-Type": []string{"application/json"}},
		body:   []byte(body),
	}
	s.replies = next
}

// SetIgnoreAPIVersion makes documents.info answer the unwrapped shape whatever X-API-Version the client sends.
func (s *Stub) SetIgnoreAPIVersion(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ignoreVersion = on
}

// SetEditorOnlyDocuments makes documents.info answer editor data with no Markdown text, as a server at version 3 does.
func (s *Stub) SetEditorOnlyDocuments(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.editorOnly = on
}

// SetAccount sets the user and the team that auth.info answers with. An empty field keeps the default.
func (s *Stub) SetAccount(user, team string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = Account{User: user, Team: team}
}

// Requests returns a copy of every request received, in arrival order.
func (s *Stub) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Calls counts the requests received for one Outline method, such as "collections.list".
func (s *Stub) Calls(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.requests {
		if r.Method == method {
			n++
		}
	}
	return n
}

// PeakInFlight returns the most requests that were inside the handler at once.
func (s *Stub) PeakInFlight() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peak
}

// Close stops a started Stub and releases any request that is still hanging.
// It is safe to call more than once, and it does nothing for an in-process Stub.
func (s *Stub) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.server != nil {
			s.server.Close()
		}
	})
}

// ServeHTTP answers one request on the TLS listener.
func (s *Stub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rep := s.handle(r, r.TLS != nil)
	switch {
	case rep.hang:
		select {
		case <-r.Context().Done():
		case <-s.done:
		}
	case rep.drop:
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			return
		}
		_ = conn.Close()
	default:
		for name, values := range rep.header {
			w.Header()[name] = values
		}
		w.WriteHeader(rep.status)
		_, _ = w.Write(rep.body)
	}
}

// handle records the request, decides the answer, and returns it. secure says whether the
// request arrived over TLS. It holds the lock only to read or write state, so overlapping
// requests remain visible to PeakInFlight.
func (s *Stub) handle(r *http.Request, secure bool) reply {
	method := strings.TrimPrefix(r.URL.Path, apiPrefix)
	if !strings.HasPrefix(r.URL.Path, apiPrefix) {
		method = ""
	}
	in, raw, bodyErr := readBody(r)
	auth := r.Header.Get("Authorization")
	version := apiVersionOf(r.Header.Get("X-API-Version"))

	s.mu.Lock()
	idx := len(s.requests)
	s.requests = append(s.requests, Request{
		Method:        method,
		HTTPMethod:    r.Method,
		Secure:        secure,
		Path:          r.URL.Path,
		Host:          r.Host,
		Authorization: auth,
		Offset:        in.Offset,
		Limit:         in.Limit,
		ID:            in.ID,
		At:            s.now(),
		ContentType:   r.Header.Get("Content-Type"),
		APIVersion:    r.Header.Get("X-API-Version"),
		ClientVersion: r.Header.Get("X-Client-Version"),
		Body:          raw,
	})
	s.inFlight++
	if s.inFlight > s.peak {
		s.peak = s.inFlight
	}
	snap := s.snapshot()
	s.mu.Unlock()

	rep := snap.respond(call{
		httpMethod: r.Method,
		method:     method,
		in:         in,
		bodyErr:    bodyErr,
		auth:       auth,
		version:    version,
	})

	s.mu.Lock()
	s.requests[idx].Status = rep.status
	s.inFlight--
	s.mu.Unlock()
	return rep
}

// snapshot copies the state a reply needs. It consumes one use of a limited fault. The caller holds s.mu.
func (s *Stub) snapshot() snapshot {
	sn := snapshot{
		collections:   s.collections,
		documents:     s.documents,
		fullPages:     s.fullPages,
		unoffered:     s.unoffered,
		replies:       s.replies,
		ignoreVersion: s.ignoreVersion,
		editorOnly:    s.editorOnly,
		account:       s.accountOrDefault(),
	}
	if s.faultLeft != 0 {
		if s.faultLeft > 0 {
			s.faultLeft--
		}
		if s.fault != (Fault{}) {
			sn.fault = s.fault
		}
	}
	return sn
}

// inProcess is the RoundTripper of an in-process Stub.
type inProcess struct {
	stub *Stub
}

// RoundTrip answers one request synchronously, without a socket. A request to a host that is neither a loopback address
// nor a .invalid name is refused and counted, so the in-process stub never stands in for a real wiki.
func (t inProcess) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		defer req.Body.Close()
	}
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	if !answerableHost(req.URL.Hostname()) {
		t.stub.mu.Lock()
		t.stub.refused++
		t.stub.mu.Unlock()
		return nil, errRefusedHost
	}
	rep := t.stub.handle(req, req.URL.Scheme == "https")
	switch {
	case rep.hang:
		return nil, context.DeadlineExceeded
	case rep.drop:
		return nil, errConnectionDropped
	}
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", rep.status, http.StatusText(rep.status)),
		StatusCode:    rep.status,
		Header:        rep.header,
		Body:          io.NopCloser(bytes.NewReader(rep.body)),
		ContentLength: int64(len(rep.body)),
		Request:       req,
	}, nil
}

// readBody decodes the JSON body of a request. It returns the fields the replay uses, and every key of the body. A
// missing body is an empty request.
func readBody(r *http.Request) (requestBody, map[string]any, error) {
	var in requestBody
	if r.Body == nil {
		return in, nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err != nil {
		return in, nil, fmt.Errorf("outlinestub: read request body: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return in, nil, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return in, nil, fmt.Errorf("outlinestub: decode request body: %w", err)
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return in, raw, fmt.Errorf("outlinestub: decode request body: %w", err)
	}
	return in, raw, nil
}

// requestBody is the JSON body the client sends to an Outline method.
type requestBody struct {
	Offset       int    `json:"offset"`
	Limit        int    `json:"limit"`
	ID           string `json:"id"`
	Query        string `json:"query"`
	CollectionID string `json:"collectionId"`
}

// apiVersionOf reads the X-API-Version header. A missing or non-numeric value is version 0, the server's default.
func apiVersionOf(header string) int {
	n, err := strconv.Atoi(strings.TrimSpace(header))
	if err != nil {
		return 0
	}
	return n
}

// accountOrDefault returns the account the stub answers with. The caller holds s.mu.
func (s *Stub) accountOrDefault() Account {
	acc := defaultAccount
	if s.account.User != "" {
		acc.User = s.account.User
	}
	if s.account.Team != "" {
		acc.Team = s.account.Team
	}
	return acc
}
