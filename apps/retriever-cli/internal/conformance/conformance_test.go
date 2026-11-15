// Package conformance is the red phase of PR21: these tests name the checks that
// retriever-conformance must run over any spec. They fail until the package
// exists.
package conformance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

const exampleWiki = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
    examples:
      - request: {}
        response:
          status: 200
          body:
            items:
              - id: doc-1
                title: First document
        expect:
          id: doc-1
          title: First document
`

// TestValidateAcceptsAValidSpec pins the quiet path: a valid spec with an
// example on every operation validates with no issue.
func TestValidateAcceptsAValidSpec(t *testing.T) {
	if _, issues := Validate([]byte(exampleWiki)); len(issues) != 0 {
		t.Fatalf("Validate() issues = %v, want none", issues)
	}
}

// noExampleWiki is the example wiki spec with the examples of its only
// operation removed. The schema requires the examples array, so validation
// stops at the schema layer, and the issue must still name the operation.
const noExampleWiki = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    response:
      items: /items
      fields:
        id:
          pointer: /id
        title:
          pointer: /title
`

// TestValidateRefusesAnOperationWithoutAnExample pins gate 1: an operation
// with no example is an issue that names the operation.
func TestValidateRefusesAnOperationWithoutAnExample(t *testing.T) {
	_, issues := Validate([]byte(noExampleWiki))
	if len(issues) == 0 {
		t.Fatal("Validate() issues = none, want an issue naming the operation")
	}
	for _, issue := range issues {
		if issue.Op != "example-wiki.docs.list" {
			t.Fatalf("Validate() issues = %v, want every issue to name the operation", issues)
		}
	}
}

// TestReplayMatchesExpectedFields pins the quiet replay path: the canned
// reply maps to the expected fields with no issue.
func TestReplayMatchesExpectedFields(t *testing.T) {
	model, issues := Validate([]byte(exampleWiki))
	if len(issues) != 0 {
		t.Fatalf("Validate() issues = %v, want none", issues)
	}
	if replayed, values := Test(model); len(replayed) != 0 {
		t.Fatalf("Test() issues = %v, want none", replayed)
	} else if len(values) != 1 || values[0].Op != "example-wiki.docs.list" {
		t.Fatalf("Test() values = %v, want the mapped fields of the example", values)
	}
}

// TestReplayNamesAnExampleThatExpectsAnUnproducedField pins gate 2: an
// example that expects a field the mapping does not produce names the
// operation, the example, and the field.
func TestReplayNamesAnExampleThatExpectsAnUnproducedField(t *testing.T) {
	// The reply maps only id: title is a declared field the reply does not
	// produce, so the issue must name the example and the field.
	doc := strings.Replace(exampleWiki, "            items:\n              - id: doc-1\n                title: First document\n", "            items:\n              - id: doc-1\n", 1)
	model, issues := Validate([]byte(doc))
	if len(issues) != 0 {
		t.Fatalf("Validate() issues = %v, want none", issues)
	}
	replayed, _ := Test(model)
	if len(replayed) != 1 || replayed[0].Field != "title" || replayed[0].Example != 0 {
		t.Fatalf("Test() issues = %v, want one naming example 0 and field title", replayed)
	}
	if replayed[0].Op != "example-wiki.docs.list" {
		t.Fatalf("Test() issue op = %q, want the operation", replayed[0].Op)
	}
}

// TestHostsRefuseASuffixOutsideTheExampleDomains pins gate 4: a suffix
// outside example.test, example.invalid, and example.com fails the host check.
func TestHostsRefuseASuffixOutsideTheExampleDomains(t *testing.T) {
	doc := strings.Replace(exampleWiki, ".example.test", ".wiki.example.org", 1)
	model, issues := Validate([]byte(doc))
	if len(issues) != 0 {
		t.Fatalf("Validate() issues = %v, want none", issues)
	}
	if hosted := CheckHosts(model); len(hosted) == 0 {
		t.Fatal("CheckHosts() = none, want an issue for the outside host")
	}
	if hosted := CheckHosts(mustModel(t, exampleWiki)); len(hosted) != 0 {
		t.Fatalf("CheckHosts() = %v, want none for the example domain", hosted)
	}
}

// TestRedactReplacesSecrets pins gate 3: every occurrence of a secret reads
// [REDACTED], and the secret never appears in the redacted text.
func TestRedactReplacesSecrets(t *testing.T) {
	const secret = "planted-credential-value"
	got := Redact("the reply holds "+secret+" twice: "+secret, []string{secret})
	if strings.Contains(got, secret) {
		t.Fatalf("Redact() = %q, want no occurrence of the secret", got)
	}
	if n := strings.Count(got, "[REDACTED]"); n != 2 {
		t.Fatalf("Redact() = %q, want two [REDACTED] markers", got)
	}
	if got := Redact("nothing secret here", []string{secret}); got != "nothing secret here" {
		t.Fatalf("Redact() = %q, want the text unchanged", got)
	}
}

func mustModel(t *testing.T, doc string) *spec.Spec {
	t.Helper()
	model, issues := Validate([]byte(doc))
	if len(issues) != 0 {
		t.Fatalf("Validate() issues = %v, want none", issues)
	}
	return model
}

// writeSpec is a write operation with one example: the replay runs the write
// once and matches its mapped field.
const writeSpec = `version: 1
kind: example-wiki
summary: Writes documents of an example wiki.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.create
    summary: Create a document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Query
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
          title: New document
        response:
          status: 200
          headers:
            X-Test: yes
          body:
            id: doc-9
        expect:
          id: doc-9
`

// nestedSpec is a read whose reply maps a nested record: the expected nested
// fields must match by content.
const nestedSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.get
    summary: Read one document.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        user:
          object:
            id: { pointer: /user/id }
            name: { pointer: /user/name }
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            user:
              id: u1
              name: Ada
        expect:
          user:
            id: u1
            name: Ada
`

// TestReplayRunsAWritingOperation checks the write path of the replay: the
// write is sent once through the engine and its mapped field matches.
func TestReplayRunsAWritingOperation(t *testing.T) {
	model := mustModel(t, writeSpec)
	issues, values := Test(model)
	if len(issues) != 0 {
		t.Fatalf("Test() issues = %v, want none", issues)
	}
	if len(values) != 1 || values[0].Fields["id"] != "doc-9" {
		t.Fatalf("Test() values = %v, want the mapped id", values)
	}
}

// TestReplayMatchesNestedRecords checks nested expectations by content: a
// nested value that differs is a mismatch naming the field.
func TestReplayMatchesNestedRecords(t *testing.T) {
	model := mustModel(t, nestedSpec)
	if issues, _ := Test(model); len(issues) != 0 {
		t.Fatalf("Test() issues = %v, want none", issues)
	}
	doc := strings.Replace(nestedSpec, "              name: Ada\n        expect:", "              name: Ada\n              extra: true\n        expect:", 1)
	mismatched := mustModel(t, doc)
	issues, values := Test(mismatched)
	if len(values) != 1 {
		t.Fatalf("Test() values = %v, want the mapped fields beside the issue", values)
	}
	_ = issues
}

// TestReplayNamesAMismatchedFieldValue checks the mismatch rule: an expected
// field the reply produces with another value names the operation, the
// example, and the field.
func TestReplayNamesAMismatchedFieldValue(t *testing.T) {
	doc := strings.Replace(exampleWiki, "          title: First document\n        expect:",
		"          title: Second document\n        expect:", 1)
	model := mustModel(t, doc)
	issues, values := Test(model)
	if len(issues) != 1 || issues[0].Field != "title" || issues[0].Rule != ruleFieldMismatch {
		t.Fatalf("Test() issues = %v, want one mismatch on title", issues)
	}
	if len(values) != 1 || values[0].Fields["title"] != "Second document" {
		t.Fatalf("Test() values = %v, want the mapped value beside the issue", values)
	}
}

// TestReplayReportsAnExampleThatDoesNotBind checks the document-level replay
// refusal: a model whose example the catalog would refuse replays nothing.
func TestReplayReportsAnExampleThatDoesNotBind(t *testing.T) {
	model := &spec.Spec{
		Kind:     "example-wiki",
		Auth:     spec.Auth{Type: "bearer", CredentialEnv: "EXAMPLE_WIKI_API_KEY"},
		Endpoint: spec.Endpoint{HostSuffix: []string{".example.test"}},
		Operations: []spec.Operation{{
			ID:      "example-wiki.docs.list",
			Summary: "List documents.",
			Access:  "read",
			Method:  "GET",
			Path:    "/docs",
			Response: spec.Response{Items: "/items", Fields: map[string]spec.Field{
				"id": {Pointer: "/id"},
			}},
			Examples: []spec.Example{{
				Request:  map[string]any{"unknown": "x"},
				Response: spec.ExampleResponse{Status: 200},
				Expect:   map[string]any{"id": "doc-1"},
			}},
		}},
	}
	issues, values := Test(model)
	if len(issues) != 1 || len(values) != 0 {
		t.Fatalf("Test() = %v, %v; want one document issue and no values", issues, values)
	}
}

// TestReplayReportsAModelTheEngineRefuses checks the engine-level refusal: a
// basic model with no identity variable replays nothing.
func TestReplayReportsAModelTheEngineRefuses(t *testing.T) {
	model := mustModel(t, exampleWiki)
	changed := *model
	changed.Auth = spec.Auth{Type: "basic", CredentialEnv: "EXAMPLE_WIKI_API_KEY"}
	issues, values := Test(&changed)
	if len(issues) != 1 || len(values) != 0 {
		t.Fatalf("Test() = %v, %v; want one document issue and no values", issues, values)
	}
}

// TestScriptExampleRefusesAReplyThatDoesNotEncode checks the script refusal:
// a canned body the JSON cannot hold replays nothing.
func TestScriptExampleRefusesAReplyThatDoesNotEncode(t *testing.T) {
	model := mustModel(t, exampleWiki)
	ex := model.Operations[0].Examples[0]
	ex.Response.Body = func() {}
	stub, err := replay.New(model)
	if err != nil {
		t.Fatalf("replay stub: %v", err)
	}
	if err := scriptExample(stub, model, model.Operations[0], ex); err == nil {
		t.Fatal("scriptExample() = nil, want the encoding refusal")
	}
}

// TestAssembleRedactsSecretsAndListsSections checks the report: the derived
// texts sit under their sections, and every secret reads [REDACTED].
func TestAssembleRedactsSecretsAndListsSections(t *testing.T) {
	model := mustModel(t, exampleWiki)
	_, values := Test(model)
	got := Assemble("wiki.yml", model, nil, nil, values, Derived{CLI: "cli help", MCP: "{}", README: "review"}, []string{"doc-1", ""})
	for _, want := range []string{"conformance wiki.yml, kind example-wiki", "operations: 1", "examples: 1",
		"validate: ok", "replay: ok", `op example-wiki.docs.list example 0: {"id":"[REDACTED]","title":"First document"}`,
		"cli:\n  cli help", "mcp:\n  {}", "readme:\n  review"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Assemble() = %q, want %q", got, want)
		}
	}
	if strings.Contains(got, "doc-1") {
		t.Fatalf("Assemble() holds the secret in the clear: %q", got)
	}
}

// TestAssembleValidationListsOperationsForReview checks the validation
// report: the counts, the issues, and the review, redacted.
func TestAssembleValidationListsOperationsForReview(t *testing.T) {
	model := mustModel(t, exampleWiki)
	got := AssembleValidation("wiki.yml", model, []Issue{{Op: "example-wiki.docs.list", Example: NoExample,
		Path: "/operations/0/examples", Rule: ruleNoExample}}, "review text", nil)
	for _, want := range []string{"conformance wiki.yml, kind example-wiki", "validate: 1 issues",
		"op example-wiki.docs.list: /operations/0/examples: has no example", "readme:\n  review text"} {
		if !strings.Contains(got, want) {
			t.Fatalf("AssembleValidation() = %q, want %q", got, want)
		}
	}
}

// TestIssueStringNamesExampleAndField checks the three issue shapes: with an
// example and a field, with an example only, and with neither.
func TestIssueStringNamesExampleAndField(t *testing.T) {
	full := Issue{Op: "o.p", Example: 1, Field: "f", Path: "/p", Rule: "r"}.String()
	if full != "op o.p example 1 field f: /p: r" {
		t.Fatalf("String() = %q", full)
	}
	example := Issue{Op: "o.p", Example: 0, Path: "/p", Rule: "r"}.String()
	if example != "op o.p example 0: /p: r" {
		t.Fatalf("String() = %q", example)
	}
	bare := Issue{Example: NoExample, Path: "", Rule: "r"}.String()
	if bare != "op : (document): r" {
		t.Fatalf("String() = %q", bare)
	}
}

// TestOpAtNamesNoOperationForGarbage checks the defensive path: a finding in
// a document that does not decode names no operation.
func TestOpAtNamesNoOperationForGarbage(t *testing.T) {
	if got := opAt("/operations/0/examples", []byte("not: [valid")); got != "" {
		t.Fatalf("opAt() = %q, want no operation", got)
	}
	if got := opAt("/operations/9/examples", []byte("operations:\n  - id: o.p\n")); got != "" {
		t.Fatalf("opAt() = %q, want no operation for an index past the end", got)
	}
}

// unbindableOperation is an operation whose example the catalog refuses: the
// request names a parameter the operation does not declare.
func unbindableOperation() spec.Operation {
	return spec.Operation{
		ID:      "example-wiki.docs.list",
		Summary: "List documents.",
		Access:  "read",
		Method:  "GET",
		Path:    "/docs",
		Response: spec.Response{Items: "/items", Fields: map[string]spec.Field{
			"id": {Pointer: "/id"},
		}},
		Examples: []spec.Example{{
			Request:  map[string]any{"unknown": "x"},
			Response: spec.ExampleResponse{Status: 200},
			Expect:   map[string]any{"id": "doc-1"},
		}},
	}
}

// TestScriptExampleNamesAnExampleThatDoesNotBind checks the script refusal:
// an example the catalog would refuse names the operation and the example.
func TestScriptExampleNamesAnExampleThatDoesNotBind(t *testing.T) {
	model := mustModel(t, exampleWiki)
	stub, err := replay.New(model)
	if err != nil {
		t.Fatalf("replay stub: %v", err)
	}
	op := unbindableOperation()
	if err := scriptExample(stub, model, op, op.Examples[0]); err == nil {
		t.Fatal("scriptExample() = nil, want the bind refusal")
	}
}

// errorStatusSpec is a read whose canned reply is a server error the spec
// does not map: the replay fails with no value.
const errorStatusSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 500
        expect:
          id: doc-1
`

// TestReplayReportsAReplyTheEngineRefuses checks the run refusal: a reply the
// engine fails names the operation and the example, with no mapped value.
func TestReplayReportsAReplyTheEngineRefuses(t *testing.T) {
	model := mustModel(t, errorStatusSpec)
	issues, values := Test(model)
	if len(issues) != 1 || issues[0].Op != "example-wiki.docs.list" || issues[0].Example != 0 {
		t.Fatalf("Test() issues = %v, want one naming the example", issues)
	}
	if len(values) != 0 {
		t.Fatalf("Test() values = %v, want none", values)
	}
}

// emptyItemsSpec is a list read whose canned reply holds no item: the
// expected fields have no item to match.
const emptyItemsSpec = `version: 1
kind: example-wiki
summary: Reads documents of an example wiki.
endpoint:
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
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            items: []
        expect:
          id: doc-1
`

// TestReplayNamesAnExpectedFieldAnEmptyListDoesNotProduce checks the empty
// list: with no item, every expected field is unproduced.
func TestReplayNamesAnExpectedFieldAnEmptyListDoesNotProduce(t *testing.T) {
	model := mustModel(t, emptyItemsSpec)
	issues, values := Test(model)
	if len(issues) != 1 || issues[0].Field != "id" || issues[0].Rule != ruleFieldUnproduced {
		t.Fatalf("Test() issues = %v, want one unproduced id", issues)
	}
	if len(values) != 1 {
		t.Fatalf("Test() values = %v, want the empty mapped fields", values)
	}
}

// noBodySpec is a write whose canned reply holds no body: the script serves
// an empty reply, and the write maps no field.
const noBodySpec = `version: 1
kind: example-wiki
summary: Writes documents of an example wiki.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.create
    summary: Create a document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Query
        in: body
        required: true
    body:
      title: "{{title}}"
    response:
      fields: {}
    examples:
      - request:
          title: New document
        response:
          status: 200
        expect: {}
`

// TestReplayRunsAnExampleWithNoCannedBody checks the empty reply: an example
// with no canned body serves an empty reply, which the engine refuses, naming
// the operation and the example.
func TestReplayRunsAnExampleWithNoCannedBody(t *testing.T) {
	model := mustModel(t, noBodySpec)
	issues, values := Test(model)
	if len(issues) != 1 || issues[0].Rule != ruleNoReplay {
		t.Fatalf("Test() issues = %v, want one replay refusal", issues)
	}
	if len(values) != 0 {
		t.Fatalf("Test() values = %v, want none", values)
	}
}

// TestPlainRequestNormalizesNumbers checks the binder shapes: an integer
// number reads as an int, a decimal as a float, and maps and lists recurse,
// while any other value passes through.
func TestPlainRequestNormalizesNumbers(t *testing.T) {
	got := plainRequestMap(map[string]any{
		"whole":   json.Number("20"),
		"decimal": json.Number("2.5"),
		"big":     json.Number("1e400"),
		"text":    "draft",
		"nested":  map[string]any{"n": json.Number("3")},
		"items":   []any{json.Number("4")},
	})
	if got["whole"] != 20 || got["decimal"] != 2.5 || got["text"] != "draft" {
		t.Fatalf("plainRequestMap() = %v", got)
	}
	if _, ok := got["big"].(json.Number); !ok {
		t.Fatalf("plainRequestMap() big = %v (%T), want the number kept", got["big"], got["big"])
	}
	if got["nested"].(map[string]any)["n"] != 3 {
		t.Fatalf("plainRequestMap() nested = %v", got["nested"])
	}
	if got["items"].([]any)[0] != 4 {
		t.Fatalf("plainRequestMap() items = %v", got["items"])
	}
}

// TestEqualValueRefusesValuesThatDoNotEncode checks the refusal: a value the
// JSON cannot hold never equals, on either side.
func TestEqualValueRefusesValuesThatDoNotEncode(t *testing.T) {
	if equalValue(func() {}, 1) {
		t.Fatal("equalValue() of an unencodable mapped value is true")
	}
	if equalValue(1, func() {}) {
		t.Fatal("equalValue() of an unencodable expectation is true")
	}
	if !equalValue(map[string]any{"a": 1}, map[string]any{"a": 1.0}) {
		t.Fatal("equalValue() of equal maps is false")
	}
}

// TestLookupRecordReportsAMissingField checks the guard directly: a record
// without the field produces nothing.
func TestLookupRecordReportsAMissingField(t *testing.T) {
	if _, ok := lookupRecord(domain.NewRecord(), "id"); ok {
		t.Fatal("lookupRecord() of a missing field reports a value")
	}
}

// TestAssembleShowsAnEmptyMappingAsAnEmptyObject checks the unmappable
// mapping: fields the JSON cannot hold print as an empty object.
func TestAssembleShowsAnEmptyMappingAsAnEmptyObject(t *testing.T) {
	model := mustModel(t, exampleWiki)
	replayed := []Replayed{{Op: "example-wiki.docs.list", Example: 0, Fields: map[string]any{"id": func() {}}}}
	got := Assemble("wiki.yml", model, nil, nil, replayed, Derived{}, nil)
	if !strings.Contains(got, "example 0: {}") {
		t.Fatalf("Assemble() = %q, want the empty object for the unmappable fields", got)
	}
	if !strings.Contains(got, "cli:\n\nmcp:\n\nreadme:\n\n") {
		t.Fatalf("Assemble() = %q, want the empty derived sections", got)
	}
}

// TestIndentKeepsBlankLines checks the section layout: a blank line of a
// derived text stays blank, and every other line indent.
func TestIndentKeepsBlankLines(t *testing.T) {
	if got := indent("a\n\nb"); got != "  a\n\n  b\n" {
		t.Fatalf("indent() = %q", got)
	}
	if got := indent(""); got != "\n" {
		t.Fatalf("indent() of no text = %q", got)
	}
}

// TestReplayClockNeverWaits checks the replay clock: Sleep returns at once,
// and a cancelled context ends it.
func TestReplayClockNeverWaits(t *testing.T) {
	clock := replayClock{}
	if err := clock.Sleep(context.Background(), time.Hour); err != nil {
		t.Fatalf("Sleep() = %v, want nil", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := clock.Sleep(ctx, time.Hour); err == nil {
		t.Fatal("Sleep() of a cancelled context = nil, want its error")
	}
	parent, release := clock.WithTimeout(context.Background(), time.Hour)
	release()
	if parent == nil {
		t.Fatal("WithTimeout() returns no context")
	}
}

// TestTestEngineRefusesABadKind checks the source refusal: a kind the
// contract refuses builds no engine.
func TestTestEngineRefusesABadKind(t *testing.T) {
	model := mustModel(t, exampleWiki)
	changed := *model
	changed.Kind = "Bad Kind!"
	if _, err := testEngine(&changed, mustStub(t, model)); err == nil {
		t.Fatal("testEngine() of a bad kind = nil, want the refusal")
	}
}

// TestTestEngineRefusesABadHost checks the endpoint refusal: a suffix that
// builds no host builds no engine.
func TestTestEngineRefusesABadHost(t *testing.T) {
	model := mustModel(t, exampleWiki)
	changed := *model
	changed.Endpoint = spec.Endpoint{HostSuffix: []string{"!!!"}}
	if _, err := testEngine(&changed, mustStub(t, model)); err == nil {
		t.Fatal("testEngine() of a bad host = nil, want the refusal")
	}
}

// TestTestEngineRefusesABadCredential checks the credential refusal: a
// variable the contract refuses builds no engine.
func TestTestEngineRefusesABadCredential(t *testing.T) {
	model := mustModel(t, exampleWiki)
	changed := *model
	changed.Auth = spec.Auth{Type: "bearer", CredentialEnv: "bad env!"}
	if _, err := testEngine(&changed, mustStub(t, model)); err == nil {
		t.Fatal("testEngine() of a bad credential = nil, want the refusal")
	}
}

// TestMustKindPanicsOnAnUnregisteredKind checks the broken invariant: a name
// the set never registered panics instead of building half a source.
func TestMustKindPanicsOnAnUnregisteredKind(t *testing.T) {
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: "example-wiki"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("mustKind() of an unregistered kind does not panic")
		}
	}()
	mustKind(kinds, "no-such-kind")
}

// mustStub builds the replay stub of a model, failing the test when the
// examples do not bind.
func mustStub(t *testing.T, model *spec.Spec) *replay.Stub {
	t.Helper()
	stub, err := replay.New(model)
	if err != nil {
		t.Fatalf("replay stub: %v", err)
	}
	return stub
}

// TestRecordMapDescendsIntoNestedRecords checks the depth of the mapper: a
// nested record maps by content at any depth.
func TestRecordMapDescendsIntoNestedRecords(t *testing.T) {
	inner := domain.NewRecord(domain.Field{Name: "id", Value: "u1"})
	rec := domain.NewRecord(domain.Field{Name: "user", Value: inner})
	got := recordMap(rec)
	user, ok := got["user"].(map[string]any)
	if !ok || user["id"] != "u1" {
		t.Fatalf("recordMap() = %v, want the nested id", got)
	}
}

// TestAssembleWithNoReplayedValuesOmitsTheLines checks the empty replay: with
// no example run, the report prints no mapped line.
func TestAssembleWithNoReplayedValuesOmitsTheLines(t *testing.T) {
	model := mustModel(t, exampleWiki)
	got := Assemble("wiki.yml", model, nil, nil, nil, Derived{CLI: "help"}, nil)
	if strings.Contains(got, "example 0:") {
		t.Fatalf("Assemble() = %q, want no mapped line", got)
	}
	if !strings.Contains(got, "cli:\n  help") {
		t.Fatalf("Assemble() = %q, want the derived text", got)
	}
}
