package spec

import (
	"strings"
	"testing"
)

// bundledJiraStub is a bundled stub of three Jira operations. Each operation claims one catalog row of the jira kind
// with its catalog_id, and its parameters match the row exactly: the same names, the same types, and the same
// required flags. The IssueKey rows of the kind are not claimed yet: pull request 33 admitted the IssueKey
// parameter kind to the language, and the Jira port claims those rows.
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

// bundledHiddenStub is a bundled stub of one operation that no catalog row names, such as a fallback target. It
// stays off both surfaces, which is the only form such an operation may take.
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

// requireBundledValid fails the test unless ParseBundled accepts the document, and it returns the result.
func requireBundledValid(t *testing.T, doc string) Result {
	t.Helper()
	r := ParseBundled([]byte(doc))
	if !r.Valid() {
		t.Fatalf("ParseBundled has findings for a valid bundled stub:\n%s", allText(r))
	}
	if r.Spec == nil {
		t.Fatal("a valid bundled stub has no model")
	}
	return r
}

// requireBundledRefused fails the test unless ParseBundled refuses the document with the stable code and a finding
// at exactly wantPath. It returns the result.
func requireBundledRefused(t *testing.T, doc string, wantPath string) Result {
	t.Helper()
	r := ParseBundled([]byte(doc))
	if r.Valid() {
		t.Fatalf("ParseBundled is valid; want a finding at %q", wantPath)
	}
	if r.Spec != nil {
		t.Fatal("an invalid bundled stub must have no model")
	}
	for _, f := range r.Findings {
		if f.Code != CodeSpecInvalid {
			t.Fatalf("finding code = %q; want %q", f.Code, CodeSpecInvalid)
		}
	}
	if !hasFinding(r, wantPath) {
		t.Fatalf("no finding at %q; findings:\n%s", wantPath, allText(r))
	}
	return r
}

// TestParseBundledAcceptsABuiltInKindThatParseRefuses proves the split of the two entry points: the embedded entry
// accepts a bundled stub of a built-in kind with its catalog_id fields, and the user-folder entry refuses the same
// bytes at the kind.
func TestParseBundledAcceptsABuiltInKindThatParseRefuses(t *testing.T) {
	r := requireBundledValid(t, bundledJiraStub)
	if r.Spec.Kind != "jira" || len(r.Spec.Operations) != 3 {
		t.Fatalf("model = kind %q with %d operations; want jira with 3", r.Spec.Kind, len(r.Spec.Operations))
	}
	user := Parse([]byte(bundledJiraStub))
	if user.Valid() {
		t.Fatal("Parse accepts a bundled stub of a built-in kind; want a refusal at /kind")
	}
	if !hasFinding(user, "/kind") {
		t.Fatalf("no finding at /kind; findings:\n%s", allText(user))
	}
}

// TestParseRefusesACatalogIDInAUserSpec proves that the user-folder entry refuses the exception field: a spec of an
// ordinary kind that names a catalog_id is refused at the field.
func TestParseRefusesACatalogIDInAUserSpec(t *testing.T) {
	doc := strings.Replace(minimalSpec, "    path: /docs\n", "    path: /docs\n    catalog_id: example-wiki.docs.list\n", 1)
	r := Parse([]byte(doc))
	if r.Valid() {
		t.Fatal("Parse accepts a catalog_id in a user spec")
	}
	if !hasFinding(r, "/operations/0/catalog_id") {
		t.Fatalf("no finding at /operations/0/catalog_id; findings:\n%s", allText(r))
	}
}

// TestBundledParamsMatchTheCatalogRow proves L10 for one operation at a time: a bundled operation whose parameters
// match its catalog row in names, types, and required flags is valid, and each difference is refused at the field
// that differs.
func TestBundledParamsMatchTheCatalogRow(t *testing.T) {
	requireBundledValid(t, bundledJiraStub)
	requireBundledValid(t, bundledHiddenStub)
	cases := []struct {
		name string
		doc  string
		path string
	}{
		{"a parameter of another type", strings.Replace(bundledJiraStub,
			"      - name: limit\n        type: Int", "      - name: limit\n        type: Text", 1),
			"/operations/1/params/1/type"},
		{"a parameter that flips required", strings.Replace(bundledJiraStub,
			"      - name: limit\n        type: Int\n        in: query\n        required: false",
			"      - name: limit\n        type: Int\n        in: query\n        required: true", 1),
			"/operations/1/params/1/required"},
		{"a parameter the row does not name", strings.Replace(bundledJiraStub,
			"      - name: jql\n        type: Query", "      - name: jqlx\n        type: Query", 1),
			"/operations/2/params/0/name"},
		{"a row parameter the operation misses", strings.Replace(bundledJiraStub,
			"      - name: offset\n        type: Offset\n        in: query\n        required: false\n        default: 0\n", "", 1),
			"/operations/1/params"},
		{"a catalog id of another kind", strings.Replace(bundledJiraStub,
			"    catalog_id: jira.projects.list", "    catalog_id: kiwi.products.list", 1),
			"/operations/1/catalog_id"},
		{"a catalog id of no row", strings.Replace(bundledJiraStub,
			"    catalog_id: jira.projects.list", "    catalog_id: jira.no-such-operation", 1),
			"/operations/1/catalog_id"},
		{"a catalog id that another operation already claims", strings.Replace(bundledJiraStub,
			"    catalog_id: jira.issues.search", "    catalog_id: jira.whoami", 1),
			"/operations/2/catalog_id"},
		{"an operation with no row that stays on the command line", strings.Replace(bundledHiddenStub,
			"    path: /export\n    cli: false", "    path: /export", 1),
			"/operations/0/cli"},
		{"an operation with no row that stays on the tool list", strings.Replace(bundledHiddenStub,
			"    cli: false\n    mcp: false", "    cli: false", 1),
			"/operations/0/mcp"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireBundledRefused(t, c.doc, c.path)
		})
	}
}
