package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// TestSpecMayOnlyLowerADefault checks the rule of the limits: a spec value that is lower wins, and any other value keeps
// the default.
func TestSpecMayOnlyLowerADefault(t *testing.T) {
	if got := capDuration(30*time.Second, 0); got != 30*time.Second {
		t.Fatalf("no spec value = %v, want the default", got)
	}
	if got := capDuration(30*time.Second, 5); got != 5*time.Second {
		t.Fatalf("a lower spec value = %v, want 5s", got)
	}
	if got := capDuration(30*time.Second, 90); got != 30*time.Second {
		t.Fatalf("a higher spec value = %v, want the default", got)
	}
	if got := capInt(10, 0); got != 10 {
		t.Fatalf("capInt with no value = %d", got)
	}
	if got := capInt(10, 4); got != 4 {
		t.Fatalf("capInt lower = %d", got)
	}
	if got := capInt(10, 40); got != 10 {
		t.Fatalf("capInt higher = %d", got)
	}
}

// TestBindsHostNeedsOneMatchingSuffix checks the host rule of the spec: no suffix binds any host, and otherwise one suffix
// must match.
func TestBindsHostNeedsOneMatchingSuffix(t *testing.T) {
	if !bindsHost(nil, "docs.example.test") {
		t.Fatal("a spec with no suffix refused a host")
	}
	if !bindsHost([]string{".example.com", ".example.test"}, "docs.example.test") {
		t.Fatal("a matching suffix was refused")
	}
	if bindsHost([]string{".example.test"}, "docs.example.invalid") {
		t.Fatal("a host outside the suffix was accepted")
	}
}

// TestParseRetryAfterReadsWholeSeconds checks the wait value of a 429 reply.
func TestParseRetryAfterReadsWholeSeconds(t *testing.T) {
	cases := []struct {
		value string
		want  int
		ok    bool
	}{
		{"0", 0, true},
		{" 12 ", 12, true},
		{"", 0, false},
		{"-1", 0, false},
		{"soon", 0, false},
		{"99999999999999999999999", maxRetryAfter + 1, true},
	}
	for _, c := range cases {
		got, ok := parseRetryAfter(c.value)
		if got != c.want || ok != c.ok {
			t.Fatalf("parseRetryAfter(%q) = %d, %v; want %d, %v", c.value, got, ok, c.want, c.ok)
		}
	}
}

// TestClassifyMapsEachStatusToItsSentinel checks the reply status against the closed errors of the connectors.
func TestClassifyMapsEachStatusToItsSentinel(t *testing.T) {
	plain := spec.Operation{Method: "GET"}
	cases := []struct {
		name   string
		status int
		want   error
	}{
		{"a redirect", 302, domain.ErrRedirected},
		{"an unauthorized reply", 401, domain.ErrKeyRejected},
		{"a forbidden reply", 403, domain.ErrAccessDenied},
		{"a bad request", 400, domain.ErrBadRequest},
		{"a not found", 404, domain.ErrNotFound},
		{"a server error", 503, domain.ErrUpstreamFailure},
		{"an informational reply", 102, domain.ErrUnexpectedResponse},
		{"a conflict", 409, domain.ErrUnexpectedResponse},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, retry, err := classify(plain, c.status, "")
			if !errors.Is(err, c.want) || retry {
				t.Fatalf("classify(%d) = %v, retry %v; want %v", c.status, err, c.want, retry)
			}
		})
	}
	if _, retry, err := classify(plain, 200, ""); err != nil || retry {
		t.Fatalf("classify(200) = %v, retry %v; want no error", err, retry)
	}
	if wait, retry, err := classify(plain, 429, "3"); err != nil || !retry || wait != 3*time.Second {
		t.Fatalf("classify(429, 3) = %v, %v, %v; want a 3s retry", wait, retry, err)
	}
}

// TestClassifyHonoursTheSpecErrorMap checks that a status the spec names gets the reason it names, and that a redirect
// stays refused even when the spec maps it.
func TestClassifyHonoursTheSpecErrorMap(t *testing.T) {
	o := spec.Operation{Method: "GET", Response: spec.Response{Errors: map[string]string{
		"404": "not_found",
		"409": "document_conflict",
		"410": "operation_unavailable",
		"302": "not_found",
		"418": "no_such_reason",
	}}}
	cases := map[int]error{
		404: domain.ErrNotFound,
		409: domain.ErrDocumentConflict,
		302: domain.ErrRedirected,
		418: domain.ErrUnexpectedResponse,
	}
	for status, want := range cases {
		if _, _, err := classify(o, status, ""); !errors.Is(err, want) {
			t.Fatalf("classify(%d) with the spec = %v, want %v", status, err, want)
		}
	}
	var unavailable domain.OperationUnavailableError
	if _, _, err := classify(o, 410, ""); !errors.As(err, &unavailable) || unavailable.Method != "GET" {
		t.Fatalf("classify(410) = %v, want OperationUnavailableError for GET", err)
	}
	for reason, want := range map[string]error{
		"key_rejected":        domain.ErrKeyRejected,
		"access_denied":       domain.ErrAccessDenied,
		"rate_limited":        domain.ErrSourceRateLimited,
		"unexpected_response": domain.ErrUnexpectedResponse,
		"upstream_failure":    domain.ErrUpstreamFailure,
		"unresolved_name":     domain.ErrUnresolvedName,
	} {
		if err := reasonError(reason, "GET"); !errors.Is(err, want) {
			t.Fatalf("reasonError(%s) = %v, want %v", reason, err, want)
		}
	}
}

// TestTransportErrorMapsTimeoutsAndRedactsTheText checks the mapping of a failed exchange.
func TestTransportErrorMapsTimeoutsAndRedactsTheText(t *testing.T) {
	live := context.Background()
	if err := transportError(live, context.DeadlineExceeded, "k"); !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("a deadline = %v, want ErrTimedOut", err)
	}
	if err := transportError(live, ErrNetworkRefused, "k"); !errors.Is(err, ErrNetworkRefused) {
		t.Fatalf("a refused network = %v, want ErrNetworkRefused", err)
	}
	cancelled, cancel := context.WithCancel(live)
	cancel()
	if err := transportError(cancelled, errors.New("x"), "k"); !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled caller = %v, want context.Canceled", err)
	}
	err := transportError(live, errors.New("failed with secret-value here"), "secret-value")
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("an unknown failure = %v, want ErrUnreachable", err)
	}
	if got := err.Error(); got != domain.ErrUnreachable.Error()+": failed with [REDACTED] here" {
		t.Fatalf("the text = %q, want the redacted transport text", got)
	}
	if got := redact("no secret here", ""); got != "no secret here" {
		t.Fatalf("redact with no secret changed the text: %q", got)
	}
}

// TestExampleTargetRendersAnExampleOfAReadOnly checks the target a replayed example produces, and the refusal of an
// example whose request does not bind.
func TestExampleTargetRendersAnExampleOfAReadOnly(t *testing.T) {
	o := spec.Operation{
		ID:     "example-wiki.docs.list",
		Access: "read",
		Method: "GET",
		Path:   "/docs",
		Params: []spec.Param{{Name: "limit", Type: "Limit", In: "query", Default: 20}},
	}
	s := &spec.Spec{Kind: "example-wiki", Endpoint: spec.Endpoint{BasePath: "/v1"}}
	method, target, err := ExampleTarget(s, o, map[string]any{"limit": 20})
	if err != nil || method != "GET" || target != "/v1/docs?limit=20" {
		t.Fatalf("ExampleTarget = %q %q, %v", method, target, err)
	}
	if _, _, err := ExampleTarget(s, o, map[string]any{"limit": "many"}); err == nil {
		t.Fatal("an example with a text limit was rendered")
	}
}

// TestCheckSpecShapeNamesTheFirstFault checks that a model the spec would refuse gives a finding that names the path.
func TestCheckSpecShapeNamesTheFirstFault(t *testing.T) {
	s := &spec.Spec{Operations: []spec.Operation{{Path: "/docs/x{id}"}}}
	err := checkSpecShape(s)
	var invalid *spec.InvalidError
	if !errors.As(err, &invalid) || len(invalid.Findings) != 1 || invalid.Findings[0].Path != "/operations/0/path" {
		t.Fatalf("checkSpecShape = %v, want one finding at /operations/0/path", err)
	}
	if err := checkSpecShape(&spec.Spec{Operations: []spec.Operation{{Path: "/docs"}}}); err != nil {
		t.Fatalf("a plain path was refused: %v", err)
	}
}
