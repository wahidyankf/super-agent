package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// echoYAML is a read of one issue with the key echo: the reply value at /key must equal the issueKey argument. It
// names example hosts only.
const echoYAML = `version: 1
kind: example-wiki
summary: Reads one issue with a key echo.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.issues.get
    summary: Read one issue.
    access: read
    method: GET
    path: /issues/{issueKey}
    params:
      - name: issueKey
        type: IssueKey
        in: path
        required: true
    response:
      echo: { param: issueKey, pointer: /key }
      fields:
        key:
          pointer: /key
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
        expect:
          key: EXAMPLE-12
`

// TestKeyEchoMismatchIsUnexpected checks that a reply held for the wrong key fails: the value at the echo pointer
// must equal the request argument exactly, so a planted key for another issue ends with unexpected_response.
func TestKeyEchoMismatchIsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, echoYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":"EXAMPLE-99"}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if got := engine.Code("example-wiki", err); got != "retriever.example-wiki.unexpected_response" {
		t.Fatalf("Code = %q, want retriever.example-wiki.unexpected_response", got)
	}
}

// TestKeyEchoRefusalNamesPath checks that the echo failure names the pointer path and never the values: neither the
// request key nor the planted reply key reaches the message.
func TestKeyEchoRefusalNamesPath(t *testing.T) {
	const planted = "PLANTED-999"
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, echoYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":"`+planted+`"}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if !strings.Contains(err.Error(), "/key") {
		t.Fatalf("the echo failure names no pointer path: %q", err.Error())
	}
	for _, value := range []string{planted, "EXAMPLE-12"} {
		if strings.Contains(err.Error(), value) {
			t.Fatalf("the echo failure echoes the value %q: %q", value, err.Error())
		}
	}
}

// TestKeyEchoMatchIsAccepted checks that a reply echoing the request key is served: the mapped record is returned.
func TestKeyEchoMatchIsAccepted(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, echoYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":"EXAMPLE-12"}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got := rec.Str("key"); got != "EXAMPLE-12" {
		t.Fatalf("key = %q, want EXAMPLE-12", got)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("a served echo sent %d requests, want 1", n)
	}
}

// TestKeyEchoNonTextIsUnexpected checks that an echo pointer to a non-text value fails the same way as a mismatch:
// the check reads text only, so a number at the pointer is an unexpected response naming the path.
func TestKeyEchoNonTextIsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, echoYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":12}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if !strings.Contains(err.Error(), "/key") {
		t.Fatalf("the echo failure names no pointer path: %q", err.Error())
	}
}

// TestKeyEchoMissingIsUnexpected checks that a reply with no value at the echo pointer fails naming the path.
func TestKeyEchoMissingIsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, echoYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"other":"EXAMPLE-12"}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if !strings.Contains(err.Error(), "/key") {
		t.Fatalf("the echo failure names no pointer path: %q", err.Error())
	}
}

// TestKeyEchoOnListIsRefused checks that the engine refuses a hand-built model with an echo on a list response, the
// same refusal that validation reports, before any request is sent.
func TestKeyEchoOnListIsRefused(t *testing.T) {
	s := loadSpecFrom(t, echoYAML)
	s.Operations[0].Response.Items = "/issues"
	s.Operations[0].Response.Fields = map[string]spec.Field{"key": {Pointer: "/key"}}
	stub, err := replay.New(loadSpecFrom(t, echoYAML))
	if err != nil {
		t.Fatalf("build the replay stub: %v", err)
	}
	if _, err := engine.New(s, testSource(t), stubPorts(stub, &fakeClock{})); err == nil {
		t.Fatal("an echo on a list response is accepted")
	} else if !strings.Contains(err.Error(), "/operations/0/response/echo") {
		t.Fatalf("the refusal names no echo path: %q", err.Error())
	}
}
