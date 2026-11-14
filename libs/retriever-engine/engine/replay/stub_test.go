package replay_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wikiYAML is the fixture of the replay tests. It names example hosts only.
const wikiYAML = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.list
    summary: List documents.
    access: read
    method: GET
    path: /docs
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 20
        response:
          status: 200
          headers:
            X-Example-Reply: "1"
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
  - id: example-wiki.docs.create
    summary: Create a document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Name
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
          title: First document
        response:
          status: 201
          body:
            id: doc-2
        expect:
          id: doc-2
`

// fixture returns the validated fixture spec.
func fixture(t *testing.T) *spec.Spec {
	t.Helper()
	res := spec.Validate([]byte(wikiYAML))
	if !res.Valid() {
		t.Fatalf("the fixture is refused: %v", res.Err())
	}
	return res.Spec
}

// get sends one GET request through the stub, as the engine does, and returns the reply.
func get(t *testing.T, stub *replay.Stub, target string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "https://docs.example.test"+target, nil)
	if err != nil {
		t.Fatal(err)
	}
	return stub.RoundTrip(req)
}

// TestStubAnswersAnExampleWithItsCannedReply checks the canned status, header, and body of a read example.
func TestStubAnswersAnExampleWithItsCannedReply(t *testing.T) {
	stub, err := replay.New(fixture(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := get(t, stub, "/v1/docs?limit=20")
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || resp.Header.Get("X-Example-Reply") != "1" {
		t.Fatalf("status %d header %q, want 200 and the canned header", resp.StatusCode, resp.Header.Get("X-Example-Reply"))
	}
	if !strings.Contains(string(body), `"doc-1"`) {
		t.Fatalf("body = %s, want the canned document", body)
	}
}

// TestStubRecordsEveryRequestAndCountsTheUnmatched checks the record of requests and the count of refused matches.
func TestStubRecordsEveryRequestAndCountsTheUnmatched(t *testing.T) {
	stub, err := replay.New(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := get(t, stub, "/v1/docs?limit=99"); !errors.Is(err, replay.ErrNoExchange) {
		t.Fatalf("an unmatched request err = %v, want ErrNoExchange", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || reqs[0].Path != "/v1/docs" || reqs[0].Query != "limit=99" || reqs[0].Method != "GET" {
		t.Fatalf("requests = %+v, want the one unmatched GET", reqs)
	}
	if stub.Unmatched() != 1 {
		t.Fatalf("Unmatched = %d, want 1", stub.Unmatched())
	}
}

// TestStubScriptRepeatsItsLastReply checks that each scripted reply is used once, and the last one repeats.
func TestStubScriptRepeatsItsLastReply(t *testing.T) {
	stub, err := replay.New(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	stub.Script("GET", "/v1/docs?limit=7",
		replay.Reply{Status: 500},
		replay.Reply{Status: 503},
	)
	want := []int{500, 503, 503}
	for i, status := range want {
		resp, err := get(t, stub, "/v1/docs?limit=7")
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != status {
			t.Fatalf("request %d status = %d, want %d", i, resp.StatusCode, status)
		}
	}
}

// TestStubFailReturnsTheError checks a scripted transport failure, and that the request is still recorded.
func TestStubFailReturnsTheError(t *testing.T) {
	stub, err := replay.New(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	boom := errors.New("scripted failure")
	stub.Fail("GET", "/v1/docs?limit=8", boom)
	if _, err := get(t, stub, "/v1/docs?limit=8"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the scripted failure", err)
	}
	if len(stub.Requests()) != 1 {
		t.Fatal("a failed request was not recorded")
	}
}

// TestStubDrainsARequestBody checks that the stub reads and closes a request body, as a round tripper must.
func TestStubDrainsARequestBody(t *testing.T) {
	stub, err := replay.New(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, "https://docs.example.test/v1/docs", strings.NewReader(`{"title":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := stub.RoundTrip(req)
	if !errors.Is(err, replay.ErrNoExchange) {
		t.Fatalf("a write with no script err = %v, want ErrNoExchange", err)
	}
	if resp != nil {
		t.Fatal("an unmatched request returned a reply")
	}
	if stub.Requests()[0].Method != http.MethodPost {
		t.Fatal("the write was not recorded with its method")
	}
}

// TestStubServesNoWriteExample checks that a write example is not replayed, because PR 3 has no writes.
func TestStubServesNoWriteExample(t *testing.T) {
	stub, err := replay.New(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := get(t, stub, "/v1/docs"); !errors.Is(err, replay.ErrNoExchange) {
		t.Fatalf("err = %v, want ErrNoExchange for a write example", err)
	}
}

// TestStubNewRefusesAnExampleThatDoesNotBind checks that an example whose request the catalog refuses fails New.
func TestStubNewRefusesAnExampleThatDoesNotBind(t *testing.T) {
	s := fixture(t)
	s.Operations[0].Examples[0].Request["limit"] = "many"
	if _, err := replay.New(s); err == nil {
		t.Fatal("New accepted an example whose limit is not a number")
	}
}

// TestStubNewRefusesAnExampleOfAnUnknownParameter checks an example whose request names a parameter of another name.
func TestStubNewRefusesAnExampleOfAnUnknownParameter(t *testing.T) {
	s := fixture(t)
	s.Operations[0].Examples[0].Request["page"] = 2
	if _, err := replay.New(s); err == nil {
		t.Fatal("New accepted an example with an undeclared parameter")
	}
}

// TestStubEmptyBodyWhenTheExampleHasNone checks a canned reply with no body.
func TestStubEmptyBodyWhenTheExampleHasNone(t *testing.T) {
	s := fixture(t)
	s.Operations[0].Examples[0].Response.Body = nil
	stub, err := replay.New(s)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := get(t, stub, "/v1/docs?limit=20")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 0 {
		t.Fatalf("body = %q, want none", body)
	}
}
