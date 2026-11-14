package outlinestub_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

func TestRoundTripAnswersWithTheStatusLineAndTheBody(t *testing.T) {
	s := newStub(outlinestub.WithCollections(outlinestub.NumberedCollections(2)...))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, outlineURL("collections.list"),
		strings.NewReader(`{"offset":0,"limit":25}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if resp.StatusCode != http.StatusOK || resp.Status != "200 OK" {
		t.Fatalf("status = %d %q, want 200 OK", resp.StatusCode, resp.Status)
	}
	if resp.Request != req {
		t.Fatal("the response does not name the request it answers")
	}
	if resp.ContentLength != int64(len(body)) {
		t.Fatalf("ContentLength = %d, want the body length %d", resp.ContentLength, len(body))
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if !strings.Contains(string(body), "col-002") {
		t.Fatalf("body %q lacks the second collection", body)
	}
}

func TestRoundTripClosesTheRequestBodyItRead(t *testing.T) {
	s := newStub()
	body := &closeRecorder{Reader: strings.NewReader(`{}`)}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, outlineURL("auth.info"), body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if !body.closed {
		t.Fatal("the transport did not close the request body")
	}
}

// closeRecorder is a request body that records whether it was closed.
type closeRecorder struct {
	io.Reader
	closed bool
}

func (c *closeRecorder) Close() error {
	c.closed = true
	return nil
}

func TestRoundTripAnswersARequestWithNoBody(t *testing.T) {
	s := newStub()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, outlineURL("auth.info"), nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := s.RoundTripper().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a request with no body", resp.StatusCode)
	}
	if reqs := s.Requests(); len(reqs) != 1 || reqs[0].Body != nil {
		t.Fatalf("recorded %+v, want one request with no body", reqs)
	}
}

func TestRoundTripFailsAContextThatIsAlreadyDone(t *testing.T) {
	s := newStub()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, outlineURL("auth.info"), strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := s.RoundTripper().RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
		t.Fatal("a request whose context is done was answered")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RoundTrip returned %v, want context.Canceled", err)
	}
	if n := len(s.Requests()); n != 0 {
		t.Fatalf("a cancelled request was recorded %d times, want 0", n)
	}
}

func TestRoundTripRefusesAnUnansweredHostOnlyWhenItIsNotLoopbackOrInvalid(t *testing.T) {
	s := newStub()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://[2001:db8::1]/api/auth.info", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := s.RoundTripper().RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
		t.Fatal("a request to a public IPv6 address was answered")
	}
	if err == nil {
		t.Fatal("the refusal returned no error")
	}
	if n := s.Refused(); n != 1 {
		t.Fatalf("refused = %d, want 1", n)
	}
}
