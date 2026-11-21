package unit_test

import (
	_ "embed"
	"strings"
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// exampleWiki is the bundled example connector spec: four operations, a cursor-paged list, a read of one document, a
// create, and a delete. It is embedded, because a behaviour test reads no file.
//
//go:embed testdata/example-wiki.yml
var exampleWiki string

// bundledJiraStub is a bundled spec of the built-in kind jira with three operations. Each operation claims one catalog
// row with its catalog_id, and its parameters match the row exactly.
const bundledJiraStub = `version: 1
kind: jira
summary: Reads issues of an example Jira source.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_JIRA_API_KEY
operations:
  - id: jira.auth.whoami
    summary: Show the account that signs in.
    access: read
    method: GET
    path: /whoami
    catalog_id: jira.whoami
    response:
      fields:
        name:
          pointer: /displayName
    examples:
      - request: {}
        response:
          status: 200
          body:
            displayName: Ada
        expect:
          name: Ada
  - id: jira.projects.list
    summary: List the projects of the source.
    access: read
    method: GET
    path: /projects
    catalog_id: jira.projects.list
    params:
      - name: query
        type: Query
        in: query
        required: false
      - name: limit
        type: Int
        in: query
        required: false
        default: 25
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
    response:
      fields:
        key:
          pointer: /key
    examples:
      - request:
          query: example
        response:
          status: 200
          body:
            key: EX
        expect:
          key: EX
  - id: jira.issues.search
    summary: Search the issues that match a query.
    access: read
    method: GET
    path: /search
    catalog_id: jira.issues.search
    params:
      - name: jql
        type: Query
        in: query
        required: true
      - name: limit
        type: Limit
        in: query
        required: false
        default: 50
    response:
      fields:
        key:
          pointer: /key
    examples:
      - request:
          jql: project = EX
        response:
          status: 200
          body:
            key: EX-1
        expect:
          key: EX-1
`

// bundledHiddenStub is a bundled spec of one operation that no catalog row names. It stays off both surfaces.
const bundledHiddenStub = `version: 1
kind: jira
summary: Reads issues of an example Jira source.
endpoint:
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_JIRA_API_KEY
operations:
  - id: jira.issues.export
    summary: Export the issues of a search.
    access: read
    method: GET
    path: /export
    cli: false
    mcp: false
    params:
      - name: jql
        type: Query
        in: query
        required: true
    response:
      fields:
        key:
          pointer: /key
    examples:
      - request:
          jql: project = EX
        response:
          status: 200
          body:
            key: EX-1
        expect:
          key: EX-1
`

// mutate returns base with each (old, new) pair replaced once, in order. It fails the test when a pair's old text is
// absent, so a mutation that names text the base lacks cannot pass silently.
func mutate(t *testing.T, base string, pairs ...string) string {
	t.Helper()
	for i := 0; i+1 < len(pairs); i += 2 {
		if !strings.Contains(base, pairs[i]) {
			t.Fatalf("fixture has no text %q", pairs[i])
		}
		base = strings.Replace(base, pairs[i], pairs[i+1], 1)
	}
	return base
}

// header returns the bundled example up to and including its operations key, so that a test can write its own list.
func header() string {
	const key = "operations:\n"
	return exampleWiki[:strings.Index(exampleWiki, key)+len(key)]
}

// withOps returns the bundled example with the given operations appended after its own four, so the first appended
// operation takes index 4.
func withOps(ops ...string) string {
	return exampleWiki + strings.Join(ops, "")
}

// oneOp returns a document with the header of the bundled example and one operation.
func oneOp(op string) string {
	return header() + op
}

// validate runs the user loader on a document.
func validate(doc string) spec.Result {
	return spec.Validate([]byte(doc))
}

// hasFinding reports whether the result holds a finding at exactly path.
func hasFinding(r spec.Result, path string) bool {
	for _, f := range r.Findings {
		if f.Path == path {
			return true
		}
	}
	return false
}

// text returns the finding lines of a result, one per finding, for a failure message.
func text(r spec.Result) string {
	lines := make([]string, 0, len(r.Findings))
	for _, f := range r.Findings {
		lines = append(lines, f.String())
	}
	return strings.Join(lines, "\n")
}

// requireAccepted fails the test unless the document passes every rule and has a model.
func requireAccepted(t *testing.T, doc string) spec.Result {
	t.Helper()
	r := validate(doc)
	if !r.Valid() {
		t.Fatalf("the spec is refused; want accepted. findings:\n%s", text(r))
	}
	if r.Spec == nil {
		t.Fatal("an accepted spec has no model")
	}
	return r
}

// requireRefused fails the test unless the document is refused, with no model and only the stable code.
func requireRefused(t *testing.T, doc string) spec.Result {
	t.Helper()
	r := validate(doc)
	if r.Valid() {
		t.Fatal("the spec is accepted; want refused")
	}
	if r.Spec != nil {
		t.Fatal("a refused spec has a model")
	}
	for _, f := range r.Findings {
		if f.Code != spec.CodeSpecInvalid {
			t.Fatalf("finding code = %q; want %q", f.Code, spec.CodeSpecInvalid)
		}
	}
	return r
}

// requireRefusedAt fails the test unless the document is refused with a finding at exactly path. An empty path names
// the whole document.
func requireRefusedAt(t *testing.T, doc string, path string) spec.Result {
	t.Helper()
	r := requireRefused(t, doc)
	if !hasFinding(r, path) {
		t.Fatalf("no finding at %q; findings:\n%s", path, text(r))
	}
	return r
}
