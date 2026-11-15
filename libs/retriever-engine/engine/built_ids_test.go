package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// builtIDsYAML is a read of one issue with the four built IDs: the project key, the document ID, the collection ID,
// and the browse URL, each built from the issue key at /key. It names example hosts only, and the reply carries a
// planted attacker URL that must never reach the output.
const builtIDsYAML = `version: 1
kind: example-wiki
summary: Reads one issue with built IDs.
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
      fields:
        projectKey:
          project_key: /key
        id:
          issue_id: /key
        collectionId:
          collection_id: /key
        url:
          browse_url: /key
        self:
          pointer: /self
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
            self: https://docs.example.test/rest/api/3/issue/EXAMPLE-12
        expect:
          projectKey: EXAMPLE
          id: issue-EXAMPLE-12
          collectionId: project-EXAMPLE
          url: https://docs.example.test/browse/EXAMPLE-12
`

// TestBrowseUrlIgnoresReplyUrl checks that the browse URL is built from the endpoint origin: the reply plants an
// attacker address in its own URL field, and the output still carries the endpoint-origin /browse/ URL only.
func TestBrowseUrlIgnoresReplyUrl(t *testing.T) {
	const planted = "https://attacker.example/evil"
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, builtIDsYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":"EXAMPLE-12","self":"`+planted+`"}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got := rec.Str("url"); got != "https://docs.example.test/browse/EXAMPLE-12" {
		t.Fatalf("url = %q, want the endpoint-origin browse URL", got)
	}
	if got := rec.Str("self"); got != planted {
		t.Fatalf("self = %q, want the planted reply text kept in its own field", got)
	}
}

// TestBuiltIdRefusesBadKey checks that a reply key outside the issue-key shape fails the read: the failure names
// the field and never the planted value.
func TestBuiltIdRefusesBadKey(t *testing.T) {
	const planted = "not-a-key"
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, builtIDsYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":"`+planted+`","self":"https://docs.example.test/rest/api/3/issue/x"}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if got := engine.Code("example-wiki", err); got != "retriever.example-wiki.unexpected_response" {
		t.Fatalf("Code = %q, want retriever.example-wiki.unexpected_response", got)
	}
	// Fields map in name order, so collectionId is the first built field the read checks.
	if !strings.Contains(err.Error(), "collectionId") {
		t.Fatalf("the built-ID failure names no field: %q", err.Error())
	}
	if strings.Contains(err.Error(), planted) {
		t.Fatalf("the built-ID failure echoes the value: %q", err.Error())
	}
}

// TestProjectKeyTakesPrefix checks that the project key is the text before the last hyphen: the document ID and the
// collection ID carry the same prefix forms.
func TestProjectKeyTakesPrefix(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, builtIDsYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":"EXAMPLE-12","self":"https://docs.example.test/rest/api/3/issue/EXAMPLE-12"}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if got := rec.Str("projectKey"); got != "EXAMPLE" {
		t.Fatalf("projectKey = %q, want EXAMPLE", got)
	}
	if got := rec.Str("id"); got != "issue-EXAMPLE-12" {
		t.Fatalf("id = %q, want issue-EXAMPLE-12", got)
	}
	if got := rec.Str("collectionId"); got != "project-EXAMPLE" {
		t.Fatalf("collectionId = %q, want project-EXAMPLE", got)
	}
}

// TestBuiltIdMissingKeyIsUnexpected checks that a reply with no key at the built pointer fails naming the field.
func TestBuiltIdMissingKeyIsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, builtIDsYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"self":"https://docs.example.test/rest/api/3/issue/EXAMPLE-12"}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	// Fields map in name order, so collectionId is the first built field the read checks.
	if !strings.Contains(err.Error(), "collectionId") {
		t.Fatalf("the built-ID failure names no field: %q", err.Error())
	}
}

// TestBuiltIdNonTextKeyIsUnexpected checks that a key held as another shape fails the same way: only text passes
// the shared check.
func TestBuiltIdNonTextKeyIsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, builtIDsYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":12,"self":"https://docs.example.test/rest/api/3/issue/EXAMPLE-12"}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// builtListYAML is a list read whose items each carry an issue key, with the project key built per item. It names
// example hosts only.
const builtListYAML = `version: 1
kind: example-wiki
summary: Lists issues with the project key built per item.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.issues.search
    summary: Search issues.
    access: read
    method: GET
    path: /search
    response:
      items: /issues
      fields:
        projectKey:
          project_key: /key
        id:
          issue_id: /key
    examples:
      - request: {}
        response:
          status: 200
          body:
            issues:
              - key: EXAMPLE-1
              - key: EXAMPLE-2
        expect:
          projectKey: EXAMPLE
          id: issue-EXAMPLE-1
`

// TestBuiltIdsMapInListItems checks that each item of a list builds its own IDs: the per-item key feeds the same
// shared check, so a search row carries the project part of its own key.
func TestBuiltIdsMapInListItems(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, builtListYAML))
	stub.Script("GET", "/v1/search", page(`{"issues":[{"key":"EXAMPLE-1"},{"key":"EXAMPLE-2"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.search", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	items := rec.Items()
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if got := items[0].Str("projectKey"); got != "EXAMPLE" {
		t.Fatalf("items[0].projectKey = %q, want EXAMPLE", got)
	}
	if got := items[1].Str("id"); got != "issue-EXAMPLE-2" {
		t.Fatalf("items[1].id = %q, want issue-EXAMPLE-2", got)
	}
}

// smallBudgetYAML is a read of one issue with a tight reply cap: the raw reply fits, but the built browse URL does
// not. It names example hosts only.
const smallBudgetYAML = `version: 1
kind: example-wiki
summary: Reads one browse URL under a tight reply cap.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  max_reply_bytes: 30
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
        url:
          browse_url: /key
    examples:
      - request:
          issueKey: EXAMPLE-12
        response:
          status: 200
          body:
            key: EXAMPLE-12
        expect:
          url: https://docs.example.test/browse/EXAMPLE-12
`

// TestBuiltIdOverBudgetIsTooLarge checks that a built URL past the shared output budget fails with reply_too_large:
// the raw reply fits the cap, but the endpoint-origin URL the read builds does not.
func TestBuiltIdOverBudgetIsTooLarge(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, smallBudgetYAML))
	stub.Script("GET", "/v1/issues/EXAMPLE-12", page(`{"key":"EXAMPLE-12"}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.get", domain.NewArgs(map[string]any{"issueKey": "EXAMPLE-12"}))
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
}
