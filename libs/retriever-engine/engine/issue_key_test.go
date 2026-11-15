package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// issueKeyYAML is a read with an IssueKey path parameter. It names example hosts only.
const issueKeyYAML = `version: 1
kind: example-wiki
summary: Reads one issue by its key.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 20
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

// TestIssueKeyLowercaseIsInvalid checks that a lowercase issue key is refused with invalid_document_id before any
// request: the engine binds the path parameter through the shared strict KindIssueKey check, so no new check is
// written and no request leaves.
func TestIssueKeyLowercaseIsInvalid(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, issueKeyYAML))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "example-12"}))
	if !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("err = %v, want ErrInvalidDocumentID", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused call sent %d requests", n)
	}
}

// TestIssueKeyTraversalIsInvalid checks that an issue key holding a path separator is refused the same way: traversal
// text never reaches the request path.
func TestIssueKeyTraversalIsInvalid(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, issueKeyYAML))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-1/x"}))
	if !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("err = %v, want ErrInvalidDocumentID", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused call sent %d requests", n)
	}
}

// TestIssueKeyRefusalNamesPath checks that the refusal of a bad issue key is the invalid_document_id failure of the
// path parameter, and never echoes the value: the code names the failure, the stub sees no request, and the planted
// value reaches no message.
func TestIssueKeyRefusalNamesPath(t *testing.T) {
	const planted = "not-a-key SECRET-PLANTED-VALUE must not echo"
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, issueKeyYAML))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": planted}))
	if !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("err = %v, want ErrInvalidDocumentID", err)
	}
	if got := engine.Code("example-wiki", err); got != "retriever.example-wiki.invalid_document_id" {
		t.Fatalf("Code = %q, want retriever.example-wiki.invalid_document_id", got)
	}
	if strings.Contains(err.Error(), planted) {
		t.Fatalf("the refusal echoes the value: %q", err.Error())
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused call sent %d requests", n)
	}
}

// TestIssueKeyCanonicalAccepted checks that a canonical issue key fills the path slot: the replay stub keys the canned
// reply by the rendered request, so a run proves the replay support of the IssueKey kind.
func TestIssueKeyCanonicalAccepted(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, issueKeyYAML))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("key"); got != "EXAMPLE-12" {
		t.Fatalf("key = %q, want EXAMPLE-12", got)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || reqs[0].Path != "/v1/issues/EXAMPLE-12" {
		t.Fatalf("requests = %+v, want one GET of /v1/issues/EXAMPLE-12", reqs)
	}
}
