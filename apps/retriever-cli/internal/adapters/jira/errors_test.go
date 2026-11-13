package jira

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestTheStatusMappingOfEachReply pins the table. Each row is one status of one request.
func TestTheStatusMappingOfEachReply(t *testing.T) {
	cases := []struct {
		name   string
		ep     endpoint
		status int
		want   error
	}{
		{"redirect", epMyself, 302, domain.ErrRedirected},
		{"bad request", epMyself, 400, domain.ErrBadRequest},
		{"unauthorized", epMyself, 401, domain.ErrKeyRejected},
		{"forbidden", epMyself, 403, domain.ErrAccessDenied},
		{"missing project path", epProjectSearch, 404, domain.ErrNotFound},
		{"missing issue", epIssue, 404, domain.ErrDocumentNotFound},
		{"missing issue comments", epComments, 404, domain.ErrDocumentNotFound},
		{"removed path", epMyself, 410, domain.ErrOperationUnavailable},
		{"server error", epMyself, 500, domain.ErrUpstreamFailure},
		{"bad gateway", epMyself, 502, domain.ErrUpstreamFailure},
		{"other status", epMyself, 418, domain.ErrUnexpectedResponse},
	}
	for _, c := range cases {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(c.status, `{}`), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		var out map[string]any
		err := gw.call(context.Background(), c.ep, "EXAMPLE-12", nil, nil, &out)
		if !errors.Is(err, c.want) {
			t.Fatalf("%s (status %d): err = %v, want %v", c.name, c.status, err, c.want)
		}
	}
}

// TestA2xxReplyThatIsNotJSONIsUnexpected proves a reply whose body the client cannot decode is ErrUnexpectedResponse.
func TestA2xxReplyThatIsNotJSONIsUnexpected(t *testing.T) {
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, "<html>Log in to continue</html>"), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	var out map[string]any
	if err := gw.call(context.Background(), epMyself, "", nil, nil, &out); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestParseRetryAfterReadsWholeSecondsOnly pins the Retry-After grammar: whole, non-negative seconds.
func TestParseRetryAfterReadsWholeSecondsOnly(t *testing.T) {
	cases := []struct {
		value string
		n     int
		ok    bool
	}{
		{"0", 0, true},
		{" 7 ", 7, true},
		{"60", 60, true},
		{"", 0, false},
		{"-1", 0, false},
		{"1.5", 0, false},
		{"soon", 0, false},
		{"99999999999999999999999", maxRetryAfter + 1, true},
	}
	for _, c := range cases {
		n, ok := parseRetryAfter(c.value)
		if n != c.n || ok != c.ok {
			t.Fatalf("parseRetryAfter(%q) = %d, %v; want %d, %v", c.value, n, ok, c.n, c.ok)
		}
	}
}

// TestKindErrorNamesJiraAtTheBoundary proves a sentinel that the client returns reads as Jira, so the owner rotates the right key.
func TestKindErrorNamesJiraAtTheBoundary(t *testing.T) {
	if got := kindError(domain.ErrKeyRejected).Error(); got != "Jira rejected the API key" {
		t.Fatalf("text = %q, want %q", got, "Jira rejected the API key")
	}
	if got := kindError(domain.ErrOperationUnavailable).Error(); got != "this Jira server does not offer the endpoint" {
		t.Fatalf("text = %q, want the Jira text for a removed path", got)
	}
	if !errors.Is(kindError(domain.ErrKeyRejected), domain.ErrKeyRejected) {
		t.Fatal("the wrapped sentinel is not found with errors.Is")
	}
}

// TestKindErrorWrapsOnce proves a second wrap changes nothing, and a context error is returned as it is.
func TestKindErrorWrapsOnce(t *testing.T) {
	once := kindError(domain.ErrTimedOut)
	if twice := kindError(once); twice.Error() != once.Error() {
		t.Fatalf("second wrap changed the text: %q", twice.Error())
	}
	if err := kindError(context.Canceled); !errors.Is(err, context.Canceled) || err != context.Canceled {
		t.Fatalf("a context error was wrapped: %v", err)
	}
	if kindError(nil) != nil {
		t.Fatal("nil was wrapped")
	}
}

// TestAMissingKeyStaysTypedThroughTheWrap proves a presenter still finds the variable name after the kind is added.
func TestAMissingKeyStaysTypedThroughTheWrap(t *testing.T) {
	wrapped := kindError(fmt.Errorf("jira credentials: %w", domain.MissingKeyError{Variable: keyVariable}))
	var missing domain.MissingKeyError
	if !errors.As(wrapped, &missing) || missing.Variable != keyVariable {
		t.Fatalf("errors.As lost the MissingKeyError: %v", wrapped)
	}
}

// TestEveryPublicSentinelReadsAsJiraNeverAsOutline proves that a Jira failure never says Outline. Each public sentinel is
// wrapped as a Jira error, and its text is checked.
func TestEveryPublicSentinelReadsAsJiraNeverAsOutline(t *testing.T) {
	for _, sentinel := range domain.PublicErrors() {
		text := kindError(sentinel).Error()
		if strings.Contains(text, "Outline") {
			t.Fatalf("a Jira error for %q still says Outline: %q", sentinel, text)
		}
	}
	for _, sentinel := range []error{domain.ErrHTTPSRequired, domain.ErrPageCapReached, domain.ErrOperationUnavailable} {
		if !strings.Contains(kindError(sentinel).Error(), "Jira") {
			t.Fatalf("a Jira error for %q does not name Jira: %q", sentinel, kindError(sentinel).Error())
		}
	}
}

// TestEveryStatusFailureReadsAsJira proves each client failure, wrapped at the boundary, names Jira and never Outline.
func TestEveryStatusFailureReadsAsJira(t *testing.T) {
	for _, status := range []int{302, 400, 401, 403, 404, 410, 418, 429, 500} {
		p := &probe{script: func(int, *http.Request) (*http.Response, error) {
			return reply(status, `{}`, "Retry-After", "999"), nil
		}}
		gw := probeGateway(t, p, newFakeClock(), newSecrets())
		var out map[string]any
		err := kindError(gw.call(context.Background(), epIssue, "EXAMPLE-12", nil, nil, &out))
		if err == nil || strings.Contains(err.Error(), "Outline") {
			t.Fatalf("status %d: text %v", status, err)
		}
	}
}
