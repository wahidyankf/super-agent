// Package kiwistub is a scripted Kiwi TCMS server for tests. It answers POST /json-rpc/ with JSON-RPC 2.0 replies that
// follow the envelope and the error shapes of the Kiwi adapter, from a dataset the test sets. It
// checks the Bearer key, records every request (never the key), and can answer with one fault at a time.
//
// The stub has no real-Kiwi knowledge. Its reply field names are the adapter's wire names (internal/adapters/kiwi,
// decode.go), so a field the first live read shows differently is changed in both places. The stub replays synthetic
// fixtures; it never contacts a network host.
package kiwistub

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Endpoint is the one path the stub answers, as the Kiwi adapter calls it.
const Endpoint = "/json-rpc/"

// Product is one product. Kiwi shows it as a collection.
type Product struct {
	ID   int
	Name string
}

// Plan is one test plan. Parent is 0 for a root plan.
type Plan struct {
	ID      int
	Name    string
	Product int
	Parent  int
	Text    string
}

// Legacy holds the legacy case fields that Kiwi still returns: the steps and the expected result of old cases.
type Legacy struct {
	Setup     string
	Action    string
	Effect    string
	Breakdown string
}

// Case is one test case. Plans lists the plans the case belongs to, and Sortkey orders it in each of them. Category,
// Priority, and Status are the IDs of the lookup tables.
type Case struct {
	ID       int
	Summary  string
	Text     string
	Plans    []int
	Sortkey  int
	Status   int
	Priority int
	Category int
	Legacy   Legacy
}

// Category is one test case category. A category belongs to one product.
type Category struct {
	ID      int
	Name    string
	Product int
}

// Priority is one priority. Its name field is Value, such as "P1".
type Priority struct {
	ID    int
	Value string
}

// Status is one test case status, such as CONFIRMED.
type Status struct {
	ID   int
	Name string
}

// Dataset is everything the stub holds. A zero Dataset holds nothing, so every filter returns an empty list.
type Dataset struct {
	Products   []Product
	Plans      []Plan
	Cases      []Case
	Categories []Category
	Priorities []Priority
	Statuses   []Status
}

// Fault is one failure the stub answers with, in place of its normal reply. The zero Fault is no fault, and so is a
// Fault that names no failure, such as one with only EchoKey set.
type Fault struct {
	// Status is an HTTP status to answer with, such as 401, 403, 404, 405, 429, or 500. A 429 takes RetryAfter.
	Status int
	// RetryAfter is the Retry-After header of a 429 reply, in seconds as text, such as "1".
	RetryAfter string
	// Redirect answers 302 with a Location header, which the client must not follow.
	Redirect bool
	// Location is the Location header of a Redirect reply. Empty means a fixed example host.
	Location string
	// Oversize answers 200 with a reply larger than 10 MiB.
	Oversize bool
	// NotJSON answers 200 with a body that is not JSON.
	NotJSON bool
	// WrongID answers 200 with an id that is not the id of the request.
	WrongID bool
	// NoResultError answers 200 with a JSON-RPC envelope that holds neither a result nor an error.
	NoResultError bool
	// Never does not answer. The request ends when the client gives up.
	Never bool
	// Drop closes the connection without a reply, as a network that is down does. The client sees a transport failure.
	Drop bool
	// EchoKey writes the Bearer key into the body of a Status or NotJSON reply, so a test can prove it never leaks.
	EchoKey bool
}

// Request is one request the stub received. It keeps the method and the named params, and whether the Bearer key was
// right. It never keeps the Authorization value: AuthDigest is its SHA-256 in hex, which a check compares with the digest
// of the expected header. Status is the HTTP status the stub answered, zero when it never answered.
type Request struct {
	Verb        string
	Path        string
	Host        string
	Method      string
	Params      map[string]any
	AuthOK      bool
	AuthDigest  string
	ContentType string
	Secure      bool
	Body        []byte
	Status      int
	At          time.Time
}

// system is what the stub asks of the process: the clock, the TLS key generator, the certificate signer, the listening
// socket, the file writer, and the JSON encoder of a reply. New sets the real ones. A unit test of this package sets
// fakes, so it reads no real clock, opens no socket, and writes no file.
type system struct {
	now       func() time.Time
	genKey    func(curve elliptic.Curve, random io.Reader) (*ecdsa.PrivateKey, error)
	sign      func(random io.Reader, template, parent *x509.Certificate, pub, priv any) ([]byte, error)
	listen    func(network, address string) (net.Listener, error)
	writeFile func(name string, data []byte, perm fs.FileMode) error
	encode    func(v any) ([]byte, error)
}

// wire is one request as a transport received it. The stub copies only what it keeps.
type wire struct {
	verb        string
	path        string
	host        string
	auth        string
	contentType string
	secure      bool
	body        []byte
}

// rpcFault is a JSON-RPC error the stub answers one method with.
type rpcFault struct {
	code    int
	message string
}

// Stub is a scripted Kiwi server. It is safe for use by one test at a time, and its methods are safe to call while a
// request is in flight.
type Stub struct {
	key      string
	anyKey   bool
	sys      system
	hostData map[string]Dataset

	mu          sync.Mutex
	data        Dataset
	raw         bool
	persistent  *Fault
	next        *Fault
	methodNext  map[string]Fault
	methodErrs  map[string]rpcFault
	methodFault map[string]Fault
	requests    []Request
	inFlight    int
	peak        int

	// Start sets these three: the HTTP server, its base URL, and the leaf certificate that the CA file holds.
	server *http.Server
	url    string
	leaf   *x509.Certificate
}

// Option configures a stub at construction.
type Option func(*Stub)

// WithDataset sets the dataset the stub serves.
func WithDataset(d Dataset) Option {
	return func(s *Stub) { s.data = d }
}

// WithClock sets the clock that stamps each request and dates the TLS certificate. The unit level passes its fake clock,
// so pacing is seen without a real wait. The default is time.Now.
func WithClock(now func() time.Time) Option {
	return func(s *Stub) { s.sys.now = now }
}

// WithAnyBearer makes the stub accept any non-empty Bearer credential. A scenario whose key comes from a file or from
// several sources cannot know the key when the stub starts, so the scenario checks the digest of each request's header
// instead (Request.AuthDigest). A missing or non-Bearer header is still answered with HTTP 401.
func WithAnyBearer() Option {
	return func(s *Stub) { s.anyKey = true }
}

// WithRawText turns off the HTML escaping of text. Kiwi escapes its replies, so the default is escaped, and the adapter
// unescapes once.
func WithRawText() Option {
	return func(s *Stub) { s.raw = true }
}

// New returns a stub that accepts the Bearer key given. Any other key, or none, is answered with HTTP 401.
func New(key string, opts ...Option) *Stub {
	s := &Stub{
		key: key,
		sys: system{
			now:       time.Now,
			genKey:    ecdsa.GenerateKey,
			sign:      x509.CreateCertificate,
			listen:    net.Listen,
			writeFile: os.WriteFile,
			encode:    json.Marshal,
		},
		hostData:    map[string]Dataset{},
		methodNext:  map[string]Fault{},
		methodErrs:  map[string]rpcFault{},
		methodFault: map[string]Fault{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetDataset replaces the dataset the stub serves from its next request on.
func (s *Stub) SetDataset(d Dataset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = d
}

// SetHostDataset gives the requests whose Host header is host their own dataset. A host with no dataset of its own is
// served from the dataset of SetDataset. Each host keeps the writes made to its own dataset.
func (s *Stub) SetHostDataset(host string, d Dataset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hostData[host] = d
}

// SetMethodFault answers every call of one method with f, instead of the normal reply. A one-shot fault that is pending
// takes precedence over it. The method fault takes precedence over a persistent fault set by SetFault. An empty method is
// ignored.
func (s *Stub) SetMethodFault(method string, f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.methodFault[method] = f
}

// SetFault answers every request with f, until ClearFault. A zero Fault clears it.
func (s *Stub) SetFault(f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f == (Fault{}) {
		s.persistent = nil
		return
	}
	s.persistent = &f
}

// ClearFault stops the persistent fault.
func (s *Stub) ClearFault() {
	s.SetFault(Fault{})
}

// SetFaultFirst answers the next request with f, and then serves normally.
func (s *Stub) SetFaultFirst(f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next = &f
}

// SetMethodFaultFirst answers the next call of method with f, and then serves that method normally. Other methods are not
// affected, so a write that reads first can be faulted on the write alone. An empty method is ignored.
func (s *Stub) SetMethodFaultFirst(method string, f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if method != "" {
		s.methodNext[method] = f
	}
}

// SetMethodError answers every call of method with a JSON-RPC error object of the given code and message, inside HTTP
// 200. A code of 0 clears the error of that method.
func (s *Stub) SetMethodError(method string, code int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.methodErrs[method] = rpcFault{code: code, message: message}
}

// Requests returns a copy of every request the stub received, in order.
func (s *Stub) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Calls returns how many requests named method. The count includes a request that a fault answered.
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

// PeakInFlight returns the most requests that were in flight at one time.
func (s *Stub) PeakInFlight() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peak
}

// begin and end bracket one request for the in-flight count.
func (s *Stub) begin() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight++
	if s.inFlight > s.peak {
		s.peak = s.inFlight
	}
}

func (s *Stub) end() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight--
}

// reply is the answer to one request, before it is written to a connection.
type reply struct {
	status int
	header http.Header
	body   []byte
	hold   bool
	drop   bool
}

// answer decides the reply to one request sent over a plain HTTP exchange, with no content type and no TLS.
func (s *Stub) answer(verb, path, host, auth string, body []byte) reply {
	return s.answerWire(wire{verb: verb, path: path, host: host, auth: auth, body: body})
}

// answerWire decides the reply to one request as the transport received it. It records the request first, so a faulted
// call is still counted, and then stores the status it answered.
func (s *Stub) answerWire(in wire) reply {
	if in.verb != http.MethodPost {
		return reply{status: http.StatusMethodNotAllowed, body: []byte(`{"error":"method not allowed"}`)}
	}
	if in.path != Endpoint {
		return reply{status: http.StatusNotFound, body: []byte(`{"error":"not found"}`)}
	}
	var req struct {
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
		ID     int64          `json:"id"`
	}
	parseErr := json.Unmarshal(in.body, &req)
	authOK := s.authorized(in.auth)

	s.mu.Lock()
	s.requests = append(s.requests, Request{
		Verb: in.verb, Path: in.path, Host: in.host, Method: req.Method, Params: req.Params, AuthOK: authOK,
		AuthDigest: digest(in.auth), ContentType: in.contentType, Secure: in.secure, Body: in.body, At: s.sys.now(),
	})
	index := len(s.requests) - 1
	fault := s.takeFault(req.Method)
	s.mu.Unlock()

	rep := s.decide(fault, req.ID, authOK, parseErr != nil, in.host, req.Method, req.Params)
	s.mu.Lock()
	s.requests[index].Status = rep.status
	s.mu.Unlock()
	return rep
}

// decide writes the answer to one request that the record already holds. A fault that names a failure wins over every
// normal reply; a fault that names none answers normally.
func (s *Stub) decide(fault *Fault, id int64, authOK, malformed bool, host, method string, params map[string]any) reply {
	if fault != nil {
		if rep, faulted := s.faulted(*fault, id); faulted {
			return rep
		}
	}
	if !authOK {
		return reply{status: http.StatusUnauthorized, body: []byte(`{"error":"authentication credentials were not provided"}`)}
	}
	if malformed {
		return s.rpcReply(id, nil, &rpcErr{code: -32600, message: "Invalid Request"}, s.raw)
	}
	result, rpcFailure := s.call(host, method, params)
	return s.rpcReply(id, result, rpcFailure, s.raw)
}

// authorized reports whether a request credential is accepted: the test key, or any non-empty Bearer value when the stub
// accepts any key.
func (s *Stub) authorized(auth string) bool {
	if s.anyKey {
		return len(auth) > len("Bearer ") && strings.HasPrefix(auth, "Bearer ")
	}
	return auth == "Bearer "+s.key
}

// digest returns the SHA-256 of a credential in hex, or "" for no credential. The stub keeps this, never the value.
func digest(auth string) string {
	if auth == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(auth))
	return hex.EncodeToString(sum[:])
}

// takeFault returns the fault for this request. A one-shot fault is cleared once taken, and a one-shot fault of a method
// wins over a method fault. A method fault answers every call of its method, and it wins over a persistent fault, which
// answers every other request. The caller holds the lock.
func (s *Stub) takeFault(method string) *Fault {
	if s.next != nil {
		f := *s.next
		s.next = nil
		return &f
	}
	if f, ok := s.methodNext[method]; ok && method != "" {
		delete(s.methodNext, method)
		return &f
	}
	if f, ok := s.methodFault[method]; ok && method != "" {
		return &f
	}
	return s.persistent
}

// faulted writes the reply of a fault that names a failure. The caller's id is used where the fault keeps the envelope.
// The second result is false when the fault names no failure, so the caller serves normally.
func (s *Stub) faulted(f Fault, id int64) (reply, bool) {
	header := http.Header{}
	switch {
	case f.Never:
		return reply{hold: true}, true
	case f.Drop:
		return reply{drop: true}, true
	case f.Redirect:
		location := f.Location
		if location == "" {
			location = "https://elsewhere.example" + Endpoint
		}
		header.Set("Location", location)
		return reply{status: http.StatusFound, header: header}, true
	case f.Oversize:
		big := make([]byte, 0, 10<<20+64)
		big = append(big, `{"jsonrpc":"2.0","id":`...)
		big = append(big, fmt.Sprint(id)...)
		big = append(big, `,"result":"`...)
		for len(big) < 10<<20+32 {
			big = append(big, 'x')
		}
		big = append(big, `"}`...)
		return reply{status: http.StatusOK, body: big}, true
	case f.NotJSON:
		return reply{status: http.StatusOK, body: []byte("<html>gateway page " + s.echo(f.EchoKey) + "</html>")}, true
	case f.WrongID:
		return s.rpcReply(id+1000, "ok", nil, true), true
	case f.NoResultError:
		return reply{status: http.StatusOK, body: []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%d}`, id))}, true
	case f.Status == http.StatusTooManyRequests:
		header.Set("Retry-After", f.RetryAfter)
		return reply{status: f.Status, header: header, body: []byte(`{"error":"slow down"}`)}, true
	case f.Status > 0:
		return reply{status: f.Status, body: []byte(`{"error":"refused` + s.echo(f.EchoKey) + `"}`)}, true
	}
	return reply{}, false
}

// echo returns the Bearer key as text when the fault asks for it, and "" otherwise.
func (s *Stub) echo(enabled bool) string {
	if !enabled {
		return ""
	}
	return " key " + s.key
}

// call runs one allowlisted method. The caller holds no lock. It returns exactly one of a result or an error. A forbidden
// argument name is refused as an invalid parameter, as Kiwi refuses it.
func (s *Stub) call(host, method string, params map[string]any) (any, *rpcErr) {
	if forbiddenArgument(params) {
		return nil, &rpcErr{code: -32602, message: "Invalid params"}
	}
	s.mu.Lock()
	failure, ok := s.methodErrs[method]
	s.mu.Unlock()
	if ok && failure.code != 0 {
		return nil, &rpcErr{code: failure.code, message: failure.message}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if own, ok := s.hostData[host]; ok {
		// The host has its own dataset: the call runs on it, and its writes are kept with it.
		shared := s.data
		s.data = own
		defer func() {
			s.hostData[host] = s.data
			s.data = shared
		}()
	}
	return s.dispatch(method, params)
}

// forbiddenArgument reports whether a key of params, at any depth, is one Kiwi refuses.
func forbiddenArgument(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for key, value := range x {
			switch strings.ToLower(key) {
			case "password", "token", "api_key", "apikey", "secret":
				return true
			}
			if forbiddenArgument(value) {
				return true
			}
		}
	case []any:
		for _, item := range x {
			if forbiddenArgument(item) {
				return true
			}
		}
	}
	return false
}

// rpcErr is a JSON-RPC error object.
type rpcErr struct {
	code    int
	message string
}

// rpcReply builds a 200 JSON-RPC envelope with the result or the error. Strings in a result are HTML-escaped unless the
// stub keeps raw text, as Kiwi escapes its replies. An envelope that cannot be encoded is answered with HTTP 500.
func (s *Stub) rpcReply(id int64, result any, failure *rpcErr, raw bool) reply {
	envelope := map[string]any{"jsonrpc": "2.0", "id": id}
	if failure != nil {
		envelope["error"] = map[string]any{"code": failure.code, "message": failure.message}
	} else {
		if !raw {
			result = escapeTree(result)
		}
		envelope["result"] = result
	}
	body, err := s.sys.encode(envelope)
	if err != nil {
		return reply{status: http.StatusInternalServerError}
	}
	return reply{status: http.StatusOK, body: body}
}

// escapeTree HTML-escapes every string in a result, at any depth. A dispatched result holds only maps, slices of any, and
// scalars, so no other slice type needs a case.
func escapeTree(v any) any {
	switch x := v.(type) {
	case string:
		return html.EscapeString(x)
	case map[string]any:
		out := make(map[string]any, len(x))
		for key, value := range x {
			out[key] = escapeTree(value)
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, escapeTree(item))
		}
		return out
	}
	return v
}
