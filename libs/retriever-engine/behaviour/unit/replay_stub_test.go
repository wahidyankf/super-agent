package unit_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The replay stub of these bindings answers the requests of a test. Each test sends requests straight through the stub's
// round tripper, so the binding checks the stub itself and not the engine.

// replayWriteOp is a write of the example wiki: a POST of one document, with a canned reply of 201 Created.
const replayWriteOp = `  - id: example-wiki.docs.create
    summary: Create one document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
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
          title: First
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`

// replayRequest sends a request of the method and target through the stub, and returns the reply and the error of the stub.
func replayRequest(t *testing.T, stub *replay.Stub, method, target string, body io.Reader) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(method, "https://docs.example.test"+target, body)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	return stub.RoundTrip(req)
}

// replayBody reads and closes the body of a reply, and returns its text.
func replayBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read the reply body: %v", err)
	}
	return string(b)
}

// replayTrackedBody is a request body that records whether it was read to its end and whether it was closed.
type replayTrackedBody struct {
	r      io.Reader
	read   bool
	closed bool
}

// Read reads the body and records that it was read to its end.
func (b *replayTrackedBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	if errors.Is(err, io.EOF) {
		b.read = true
	}
	return n, err
}

// Close records that the body was closed.
func (b *replayTrackedBody) Close() error {
	b.closed = true
	return nil
}

// replayFirstExample returns the first example of the first operation of a validated spec.
func replayFirstExample(s *spec.Spec) *spec.Example {
	return &s.Operations[0].Examples[0]
}

// TestReplayStub_ReadExampleAnswersWithCannedReply binds a request that matches a read example to the canned status, header,
// and body of that example.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: A request that matches a read example receives the canned status, header, and body of that example
func TestReplayStub_ReadExampleAnswersWithCannedReply(t *testing.T) {
	s := rebindSpec(t, readsListDoc)
	replayFirstExample(s).Response.Headers = map[string]string{"X-Canned": "yes"}
	stub := rebindStub(t, s)
	resp, err := replayRequest(t, stub, http.MethodGet, "/v1/docs?limit=20", nil)
	if err != nil {
		t.Fatalf("the stub fails the matching request: %v", err)
	}
	if resp.StatusCode != http.StatusOK || resp.Header.Get("X-Canned") != "yes" {
		t.Fatalf("the reply is %d with header %q, want the canned status and header", resp.StatusCode, resp.Header.Get("X-Canned"))
	}
	if body := replayBody(t, resp); !strings.Contains(body, `"doc-1"`) {
		t.Fatalf("the body is %s, want the canned body", body)
	}
}

// TestReplayStub_WriteExampleAnswersAndRecordsBody binds a write that matches a write example to the canned write reply, and
// the body that the write sent to the record.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: A write example answers its matching write and the stub records the body the write sent
func TestReplayStub_WriteExampleAnswersAndRecordsBody(t *testing.T) {
	s := rebindSpec(t, replayWriteOp)
	stub := rebindStub(t, s)
	sent := `{"title":"First"}`
	resp, err := replayRequest(t, stub, http.MethodPost, "/v1/docs", strings.NewReader(sent))
	if err != nil {
		t.Fatalf("the stub fails the matching write: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("the reply is %d, want the canned 201", resp.StatusCode)
	}
	if body := replayBody(t, resp); !strings.Contains(body, `"doc-9"`) {
		t.Fatalf("the body is %s, want the canned write reply", body)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || string(reqs[0].Body) != sent {
		t.Fatalf("the record holds %d requests, want the body the write sent", len(reqs))
	}
}

// TestReplayStub_RecordsEveryRequestAndCountsUnmatched binds one matching and one unmatched request to a record of both and
// an unmatched count of one.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: The stub records every request and counts the requests that matched no example
func TestReplayStub_RecordsEveryRequestAndCountsUnmatched(t *testing.T) {
	stub := rebindStub(t, rebindSpec(t, readsObjectDoc("")))
	if resp, err := replayRequest(t, stub, http.MethodGet, "/v1/docs/doc-1", nil); err != nil {
		t.Fatalf("the matching request fails: %v", err)
	} else {
		_ = resp.Body.Close()
	}
	if _, err := replayRequest(t, stub, http.MethodGet, "/v1/docs/other", nil); err == nil {
		t.Fatalf("the unmatched request is answered, want it refused")
	}
	if n := len(stub.Requests()); n != 2 {
		t.Fatalf("the record holds %d requests, want both", n)
	}
	if n := stub.Unmatched(); n != 1 {
		t.Fatalf("the unmatched count is %d, want one", n)
	}
}

// TestReplayStub_ScriptUsedInOrderThenLastRepeats binds a scripted reply list to its replies in order, with the last one
// repeated after the others are used.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: A scripted reply list is used once in order and its last reply repeats
func TestReplayStub_ScriptUsedInOrderThenLastRepeats(t *testing.T) {
	stub := rebindStub(t, rebindSpec(t, readsObjectDoc("")))
	stub.Script(http.MethodGet, "/v1/docs/doc-1",
		replay.Reply{Status: http.StatusOK, Body: []byte(`{"title":"one"}`)},
		replay.Reply{Status: http.StatusOK, Body: []byte(`{"title":"two"}`)},
	)
	want := []string{`"one"`, `"two"`, `"two"`}
	for i, w := range want {
		resp, err := replayRequest(t, stub, http.MethodGet, "/v1/docs/doc-1", nil)
		if err != nil {
			t.Fatalf("request %d fails: %v", i+1, err)
		}
		if body := replayBody(t, resp); !strings.Contains(body, w) {
			t.Fatalf("request %d receives %s, want %s", i+1, body, w)
		}
	}
}

// TestReplayStub_ScriptedFailureReturnsErrorAndRecords binds a scripted transport failure to the returned error, with the
// request still recorded.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: A scripted failure returns its error and the request is still recorded
func TestReplayStub_ScriptedFailureReturnsErrorAndRecords(t *testing.T) {
	stub := rebindStub(t, rebindSpec(t, readsObjectDoc("")))
	scripted := errors.New("the scripted transport failure")
	stub.Fail(http.MethodGet, "/v1/docs/doc-1", scripted)
	_, err := replayRequest(t, stub, http.MethodGet, "/v1/docs/doc-1", nil)
	if !errors.Is(err, scripted) {
		t.Fatalf("the error is %v, want the scripted error", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("the record holds %d requests, want the failed one", n)
	}
}

// TestReplayStub_NoBodyExampleSendsEmptyBody binds a matching request to an empty reply body when the example has none.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: A reply with no body on its example sends an empty body
func TestReplayStub_NoBodyExampleSendsEmptyBody(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	replayFirstExample(s).Response.Body = nil
	stub := rebindStub(t, s)
	resp, err := replayRequest(t, stub, http.MethodGet, "/v1/docs/doc-1", nil)
	if err != nil {
		t.Fatalf("the stub fails the matching request: %v", err)
	}
	if body := replayBody(t, resp); body != "" {
		t.Fatalf("the body is %q, want it empty", body)
	}
}

// TestReplayStub_ReadsAndClosesRequestBody binds a request with a body to a stub that reads the body to its end and closes it.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: The stub reads and closes the body of each request it receives
func TestReplayStub_ReadsAndClosesRequestBody(t *testing.T) {
	stub := rebindStub(t, rebindSpec(t, replayWriteOp))
	body := &replayTrackedBody{r: strings.NewReader(`{"title":"First"}`)}
	req, err := http.NewRequest(http.MethodPost, "https://docs.example.test/v1/docs", body)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	resp, err := stub.RoundTrip(req)
	if err != nil {
		t.Fatalf("the stub fails the request: %v", err)
	}
	_ = resp.Body.Close()
	if !body.read || !body.closed {
		t.Fatalf("the body was read %v and closed %v, want both", body.read, body.closed)
	}
}

// TestReplayStub_FirstOfTwoSameReadExamplesAnswers binds two read examples with the same request to the reply of the first.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: Two read examples with the same request answer with the first one
func TestReplayStub_FirstOfTwoSameReadExamplesAnswers(t *testing.T) {
	s := rebindSpec(t, readsListDoc)
	second := replayFirstExample(s)
	dup := *second
	dup.Response.Body = map[string]any{"items": []any{map[string]any{"id": "doc-9"}}}
	s.Operations[0].Examples = append(s.Operations[0].Examples, dup)
	stub := rebindStub(t, s)
	resp, err := replayRequest(t, stub, http.MethodGet, "/v1/docs?limit=20", nil)
	if err != nil {
		t.Fatalf("the stub fails the matching request: %v", err)
	}
	body := replayBody(t, resp)
	if !strings.Contains(body, `"doc-1"`) || strings.Contains(body, `"doc-9"`) {
		t.Fatalf("the reply is %s, want the first example", body)
	}
}

// TestReplayStub_ExampleRequestNotBindingRefused binds an example request that names a parameter the catalog refuses to a
// failed construction.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: A stub is refused when an example request does not bind to its operation
func TestReplayStub_ExampleRequestNotBindingRefused(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	replayFirstExample(s).Request["bogus"] = "x"
	if _, err := replay.New(s); err == nil {
		t.Fatalf("the stub is built from an example that does not bind, want a refusal")
	}
}

// TestReplayStub_ExampleBodyWithoutJSONRefused binds an example body with no JSON form to a failed construction of the stub.
// @scenario specs/libs/retriever-engine/behaviours/replay-stub.feature: A stub is refused when an example body has no JSON form
func TestReplayStub_ExampleBodyWithoutJSONRefused(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	replayFirstExample(s).Response.Body = make(chan int)
	if _, err := replay.New(s); err == nil {
		t.Fatalf("the stub is built from a body with no JSON form, want a refusal")
	}
}
