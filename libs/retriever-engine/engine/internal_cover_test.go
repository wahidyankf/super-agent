package engine

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// staticKeys is a credentials port that returns one fixed key for any name. It is never a real key.
type staticKeys struct{}

// APIKey returns the fixed test key.
func (staticKeys) APIKey(domain.CredentialRef) (domain.ApiKey, error) {
	return domain.NewApiKey("EXAMPLE_WIKI_API_KEY", "internal-key")
}

// instantClock is a clock whose waits return at once and whose time limit is a plain cancel.
type instantClock struct{}

// WithTimeout returns a cancellable context with no timer.
func (instantClock) WithTimeout(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// Sleep returns at once.
func (instantClock) Sleep(context.Context, time.Duration) error {
	return nil
}

// intPtr returns a pointer to an int, for the optional bounds of a spec parameter.
func intPtr(v int) *int {
	return &v
}

// TestUnsupportedRefusesAResolveAndAConverter checks each part of a read that a later pull request serves: a name that is
// resolved, and a Markdown converter. An operation with neither is served.
func TestUnsupportedRefusesAResolveAndAConverter(t *testing.T) {
	resolve := spec.Operation{Params: []spec.Param{{Name: "name", Resolve: &spec.Resolve{Operation: "x.y.z", Field: "id"}}}}
	if err := unsupported(resolve); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("a resolve = %v, want ErrUnsupportedOperation", err)
	}
	for _, convert := range []string{"markdown-from-html", "markdown-from-adf"} {
		conv := spec.Operation{Response: spec.Response{Fields: map[string]spec.Field{"body": {Pointer: "/body", Convert: convert}}}}
		if err := unsupported(conv); !errors.Is(err, domain.ErrUnsupportedOperation) {
			t.Fatalf("a %s converter = %v, want ErrUnsupportedOperation", convert, err)
		}
	}
	if err := unsupported(spec.Operation{}); err != nil {
		t.Fatalf("a plain operation = %v, want nil", err)
	}
}

// TestPathOKRefusesAPathThatIsNotOneSlashedPath checks the path shape that New enforces on a hand-built model.
func TestPathOKRefusesAPathThatIsNotOneSlashedPath(t *testing.T) {
	cases := map[string]bool{
		"/docs":            true,
		"/docs/{id}":       true,
		"docs":             false,
		"/docs/.hidden":    false,
		"/docs/x{id}":      false,
		"/docs/{id}/{who}": true,
	}
	for path, want := range cases {
		if got := pathOK(path); got != want {
			t.Fatalf("pathOK(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestRenderRequestRefusesAMissingSlotAndAnUnsafeHeader checks that a request is not rendered when a path slot has no
// value, or a header value is unsafe.
func TestRenderRequestRefusesAMissingSlotAndAnUnsafeHeader(t *testing.T) {
	missing := spec.Operation{Method: "GET", Path: "/docs/{id}"}
	if _, err := renderRequest("/v1", missing, domain.NewArgs(nil), nil); !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("a missing slot = %v, want ErrMissingArgument", err)
	}
	unsafe := spec.Operation{Method: "GET", Path: "/docs", Headers: map[string]string{"X-Note": "{note}"}}
	if _, err := renderRequest("/v1", unsafe, domain.NewArgs(map[string]any{"note": "a\nb"}), nil); !errors.Is(err, ErrInvalidHeader) {
		t.Fatalf("an unsafe header = %v, want ErrInvalidHeader", err)
	}
}

// TestRenderHeadersRefusesAnUnsafeConstant checks that a constant header value gets the same check as an argument.
func TestRenderHeadersRefusesAnUnsafeConstant(t *testing.T) {
	if _, err := renderHeaders(map[string]string{"X-Fixed": "a\nb"}, nil); !errors.Is(err, ErrInvalidHeader) {
		t.Fatalf("an unsafe constant = %v, want ErrInvalidHeader", err)
	}
}

// TestExampleTargetRefusesAnUnsafeHeaderValue checks that an example whose header value is unsafe does not render.
func TestExampleTargetRefusesAnUnsafeHeaderValue(t *testing.T) {
	s := &spec.Spec{Endpoint: spec.Endpoint{BasePath: "/v1"}}
	o := spec.Operation{
		ID:      "example-wiki.docs.search",
		Access:  "read",
		Method:  "GET",
		Path:    "/docs",
		Headers: map[string]string{"X-Note": "{note}"},
		Params:  []spec.Param{{Name: "note", Type: "Text", In: "query"}},
	}
	if _, _, err := ExampleTarget(s, o, map[string]any{"note": "a\nb"}); !errors.Is(err, ErrInvalidHeader) {
		t.Fatalf("ExampleTarget with an unsafe header = %v, want ErrInvalidHeader", err)
	}
}

// TestExampleTargetChecksTheBoundsOfAnIntParameter checks that the spec's minimum and maximum bind an Int parameter, and
// that a value outside them does not render.
func TestExampleTargetChecksTheBoundsOfAnIntParameter(t *testing.T) {
	s := &spec.Spec{Endpoint: spec.Endpoint{BasePath: "/v1"}}
	o := spec.Operation{
		ID:     "example-wiki.docs.count",
		Access: "read",
		Method: "GET",
		Path:   "/docs",
		Params: []spec.Param{{Name: "n", Type: "Int", In: "query", Min: intPtr(1), Max: intPtr(5)}},
	}
	method, target, err := ExampleTarget(s, o, map[string]any{"n": 3})
	if err != nil || method != "GET" || target != "/v1/docs?n=3" {
		t.Fatalf("ExampleTarget inside the bounds = %q %q, %v", method, target, err)
	}
	if _, _, err := ExampleTarget(s, o, map[string]any{"n": 9}); !errors.Is(err, domain.ErrInvalidInt) {
		t.Fatalf("ExampleTarget above the bound = %v, want ErrInvalidInt", err)
	}
}

// TestPointerValueNeedsASlashAtTheStart checks that a pointer which does not start with a slash finds nothing, which is
// how a spec's pointer without a slash is read when a hand-built model reaches the engine.
func TestPointerValueNeedsASlashAtTheStart(t *testing.T) {
	if _, found := pointerValue(map[string]any{"a": 1}, "a"); found {
		t.Fatal("a pointer without a leading slash found a value")
	}
}

// TestExchangeSendsABodyAndRefusesAnInvalidMethod checks the body and its content type on one exchange, and that a method
// the request cannot take fails before any connection.
func TestExchangeSendsABodyAndRefusesAnInvalidMethod(t *testing.T) {
	var gotType, gotBody string
	e := &Engine{
		model:    &spec.Spec{Auth: spec.Auth{Type: "bearer"}},
		ports:    Ports{Credentials: staticKeys{}, Clock: instantClock{}},
		base:     "https://docs.example.test",
		timeout:  time.Second,
		maxReply: 1 << 20,
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotType = req.Header.Get("Content-Type")
			raw, _ := io.ReadAll(req.Body)
			gotBody = string(raw)
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})},
	}
	rep, err := e.exchange(context.Background(), request{
		method: "POST", target: "/v1/docs", header: http.Header{}, body: []byte(`{"a":1}`),
	}, false)
	if err != nil || rep.status != http.StatusOK || string(rep.body) != "{}" {
		t.Fatalf("exchange = %+v, %v; want a 200 reply with the body", rep, err)
	}
	if gotType != "application/json" || gotBody != `{"a":1}` {
		t.Fatalf("the request carried type %q and body %q", gotType, gotBody)
	}
	if _, err := e.exchange(context.Background(), request{method: "BAD METHOD", target: "/v1/docs", header: http.Header{}}, false); err == nil {
		t.Fatal("a request with an invalid method was built")
	}
}

// roundTripFunc is a transport written as a function, for the internal tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls the function.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestExitStatusOfNoErrorIsZero checks that a read which succeeded exits 0.
func TestExitStatusOfNoErrorIsZero(t *testing.T) {
	if got := ExitStatus(nil); got != 0 {
		t.Fatalf("ExitStatus(nil) = %d, want 0", got)
	}
}
