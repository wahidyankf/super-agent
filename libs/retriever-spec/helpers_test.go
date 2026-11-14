package spec

import (
	_ "embed"
	"strings"
	"testing"
)

// exampleWiki is the bundled example spec of the tests. It is embedded, because an untagged test reads no file.
//
//go:embed testdata/example-wiki.yml
var exampleWiki string

// minimalSpec is the smallest document that passes every rule: one read operation, no optional field.
const minimalSpec = `version: 1
kind: example-wiki
summary: Reads example documents.
endpoint: {}
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
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// replaceOnce returns s with the first occurrence of old replaced by repl. It panics when old is absent, so a fault row
// that names text the base document lacks fails loudly instead of testing nothing.
func replaceOnce(s, old, repl string) string {
	if !strings.Contains(s, old) {
		panic("test fixture has no text " + old)
	}
	return strings.Replace(s, old, repl, 1)
}

// mutateAll applies each (old, repl) pair in order to base, and returns the result.
func mutateAll(base string, pairs ...string) string {
	for i := 0; i+1 < len(pairs); i += 2 {
		base = replaceOnce(base, pairs[i], pairs[i+1])
	}
	return base
}

// fixtureWith returns the bundled example spec with the given (old, repl) pairs applied.
func fixtureWith(pairs ...string) string {
	return mutateAll(exampleWiki, pairs...)
}

// minimalWith returns the minimal spec with the given (old, repl) pairs applied.
func minimalWith(pairs ...string) string {
	return mutateAll(minimalSpec, pairs...)
}

// hasFinding reports whether the result holds a finding at exactly path.
func hasFinding(r Result, path string) bool {
	for _, f := range r.Findings {
		if f.Path == path {
			return true
		}
	}
	return false
}

// allText returns every string a caller could print from the result: each finding, and the error text.
func allText(r Result) string {
	var b strings.Builder
	for _, f := range r.Findings {
		b.WriteString(f.String())
		b.WriteString("\n")
		b.WriteString(f.Path)
		b.WriteString("\n")
		b.WriteString(f.Rule)
		b.WriteString("\n")
	}
	if err := r.Err(); err != nil {
		b.WriteString(err.Error())
	}
	return b.String()
}

// requireValid fails the test unless the document passes every rule, and it returns the result.
func requireValid(t *testing.T, doc string) Result {
	t.Helper()
	r := Validate([]byte(doc))
	if !r.Valid() {
		t.Fatalf("a valid document has findings:\n%s", allText(r))
	}
	if r.Spec == nil {
		t.Fatal("a valid document has no model")
	}
	return r
}

// requireRefused fails the test unless the document is refused with the stable code, carries a finding at exactly
// wantPath, and shows none of the given secrets in any text. It returns the result.
func requireRefused(t *testing.T, doc []byte, wantPath string, secrets ...string) Result {
	t.Helper()
	r := Validate(doc)
	if r.Valid() {
		t.Fatalf("the document is valid; want a finding at %q", wantPath)
	}
	if r.Spec != nil {
		t.Fatal("an invalid document must have no model")
	}
	for _, f := range r.Findings {
		if f.Code != CodeSpecInvalid {
			t.Fatalf("finding code = %q; want %q", f.Code, CodeSpecInvalid)
		}
	}
	if !hasFinding(r, wantPath) {
		t.Fatalf("no finding at %q; findings:\n%s", wantPath, allText(r))
	}
	text := allText(r)
	for _, secret := range secrets {
		if secret != "" && strings.Contains(text, secret) {
			t.Fatalf("a finding shows the value %q:\n%s", secret, text)
		}
	}
	return r
}
