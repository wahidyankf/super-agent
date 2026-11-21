package unit_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The support of the unit bindings. Every name carries the rebind prefix, so the bindings of the other features of this
// package never collide with it. The doubles are in memory: no clock, file, environment, socket, or process is read.

const (
	// rebindKind is the example kind of every spec the bindings build.
	rebindKind = "example-wiki"
	// rebindCredential is the example variable that names the key.
	rebindCredential = "EXAMPLE_WIKI_API_KEY"
	// rebindKey is a fake key that the credentials double returns. It is not a real key.
	rebindKey = "fixture-key"
	// rebindIdentity is a documentation address for basic authentication.
	rebindIdentity = "reader@example.test"
)

// rebindHeader is the header of every spec the bindings build. The operations of a test follow it.
const rebindHeader = `version: 1
kind: example-wiki
summary: Example wiki reads and writes for the unit bindings.
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

// rebindSpec validates the header with the given operations, and fails the test when the document is refused.
func rebindSpec(t *testing.T, operations string) *spec.Spec {
	t.Helper()
	return rebindDoc(t, rebindHeader+operations)
}

// rebindDoc validates a whole document, and fails the test when the document is refused.
func rebindDoc(t *testing.T, doc string) *spec.Spec {
	t.Helper()
	res := spec.Validate([]byte(doc))
	if !res.Valid() {
		t.Fatalf("the bound spec is refused: %v", res.Err())
	}
	return res.Spec
}

// rebindSource returns a source of the example kind, on a documentation host, with the example credential name.
func rebindSource(t *testing.T) domain.Source {
	t.Helper()
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: rebindKind})
	if err != nil {
		t.Fatal(err)
	}
	kind, err := kinds.Kind(rebindKind)
	if err != nil {
		t.Fatal(err)
	}
	id, err := domain.NewSourceID("wiki-main")
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatal(err)
	}
	cred, err := domain.NewCredentialRef(rebindCredential)
	if err != nil {
		t.Fatal(err)
	}
	tags, err := domain.NewTagSet()
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(id, kind, tags, endpoint, cred)
}

// rebindCredentials is the credentials port of a binding. It returns the fake key for any name, and it counts the reads.
type rebindCredentials struct {
	mu    sync.Mutex
	calls int
}

// APIKey returns the fake key and counts the call.
func (c *rebindCredentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return domain.NewApiKey(ref.Name(), rebindKey)
}

// rebindIdentities is the identity port of a binding. It returns a documentation address.
type rebindIdentities struct{}

// Identity returns the documentation address.
func (rebindIdentities) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	return domain.NewIdentity(ref.Name(), rebindIdentity)
}

// rebindClock is the clock of a binding. It records each time limit and each wait, and it never reads the real clock
// or starts a timer. A binding sets sleepErr to end a wait with an error, and expired to make each request's time limit
// have already passed.
type rebindClock struct {
	mu       sync.Mutex
	limits   []time.Duration
	waits    []time.Duration
	sleepErr error
	expired  bool
}

// WithTimeout records the limit. An expired clock returns a context whose time limit has passed; any other clock returns
// a cancellable context with no timer.
func (c *rebindClock) WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	c.mu.Lock()
	c.limits = append(c.limits, d)
	expired := c.expired
	c.mu.Unlock()
	if expired {
		return rebindExpiredContext{Context: parent}, func() {}
	}
	return context.WithCancel(parent)
}

// Sleep records the wait and returns at once with the error of the clock, which is nil unless a binding set one.
func (c *rebindClock) Sleep(_ context.Context, d time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.waits = append(c.waits, d)
	return c.sleepErr
}

// recordedWaits returns a copy of the waits the engine asked for.
func (c *rebindClock) recordedWaits() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.waits...)
}

// recordedLimits returns a copy of the time limits the engine asked for.
func (c *rebindClock) recordedLimits() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.limits...)
}

// rebindExpiredContext is a context whose time limit has already passed. Its Done channel is closed and its Err is the
// deadline error, which is what a request sees once its time limit has fired.
type rebindExpiredContext struct {
	context.Context
}

// Err returns the deadline error of a passed time limit.
func (rebindExpiredContext) Err() error { return context.DeadlineExceeded }

// Done returns a channel that is already closed.
func (rebindExpiredContext) Done() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

// rebindReply is one canned reply of the in-memory transport. A reply with err fails the request. A stalled reply waits
// for the request's context to end and returns that context's error, as a request that reached its time limit does.
type rebindReply struct {
	status  int
	header  http.Header
	body    string
	err     error
	stall   bool
	bodyErr error
}

// rebindFailingReader is a reply body that fails with its error at the first read, as a body that breaks while it is
// read does.
type rebindFailingReader struct {
	err error
}

// Read fails with the error of the reader.
func (r rebindFailingReader) Read([]byte) (int, error) {
	return 0, r.err
}

// rebindJSON returns a reply with a JSON body and the given status.
func rebindJSON(status int, body string) rebindReply {
	return rebindReply{status: status, header: http.Header{"Content-Type": {"application/json"}}, body: body}
}

// rebindRequest is one request that the in-memory transport received.
type rebindRequest struct {
	method string
	url    *url.URL
	header http.Header
	body   []byte
}

// rebindTransport is an in-memory round tripper. Its reply function picks the reply of each request by the request's
// index, and every request is recorded, so a binding can check what the engine sent.
type rebindTransport struct {
	mu       sync.Mutex
	reply    func(n int, req rebindRequest) rebindReply
	requests []rebindRequest
}

// rebindReplies answers the requests in order, one reply each. The last reply repeats once the others are used up.
func rebindReplies(replies ...rebindReply) *rebindTransport {
	return &rebindTransport{reply: func(n int, _ rebindRequest) rebindReply {
		if n >= len(replies) {
			return replies[len(replies)-1]
		}
		return replies[n]
	}}
}

// rebindFunc answers each request with the reply that fn picks for its index and its contents.
func rebindFunc(fn func(n int, req rebindRequest) rebindReply) *rebindTransport {
	return &rebindTransport{reply: fn}
}

// RoundTrip records the request, reads and closes its body, and returns the reply that the function picks for it.
func (tr *rebindTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	rec := rebindRequest{method: req.Method, url: req.URL, header: req.Header.Clone(), body: body}
	tr.mu.Lock()
	n := len(tr.requests)
	tr.requests = append(tr.requests, rec)
	fn := tr.reply
	tr.mu.Unlock()
	rep := fn(n, rec)
	if rep.stall {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
	if rep.err != nil {
		return nil, rep.err
	}
	header := rep.header.Clone()
	if header == nil {
		header = http.Header{}
	}
	var reader io.Reader = strings.NewReader(rep.body)
	if rep.bodyErr != nil {
		reader = io.MultiReader(strings.NewReader(rep.body), rebindFailingReader{err: rep.bodyErr})
	}
	return &http.Response{
		StatusCode:    rep.status,
		Header:        header,
		Body:          io.NopCloser(reader),
		ContentLength: int64(len(rep.body)),
		Request:       req,
	}, nil
}

// seen returns a copy of every request the transport received, in order.
func (tr *rebindTransport) seen() []rebindRequest {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return append([]rebindRequest(nil), tr.requests...)
}

// rebindEngine builds an engine for a spec over a round tripper and a clock, and fails the test when it is refused.
func rebindEngine(t *testing.T, s *spec.Spec, rt http.RoundTripper, clock engine.Clock, opts ...engine.Option) *engine.Engine {
	t.Helper()
	ports := engine.Ports{
		Credentials: &rebindCredentials{},
		Identities:  rebindIdentities{},
		Clock:       clock,
		Transport:   rt,
	}
	eng, err := engine.New(s, rebindSource(t), ports, opts...)
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	return eng
}

// rebindStub builds the replay stub of a spec, and fails the test when an example of the spec does not bind.
func rebindStub(t *testing.T, s *spec.Spec) *replay.Stub {
	t.Helper()
	stub, err := replay.New(s)
	if err != nil {
		t.Fatalf("build the replay stub: %v", err)
	}
	return stub
}

// rebindIDs returns the id field of each item of a list record, in order.
func rebindIDs(rec domain.Record) []string {
	var ids []string
	for _, item := range rec.Items() {
		ids = append(ids, item.Str("id"))
	}
	return ids
}
