package spec

import (
	"strings"
	"testing"
)

// echoDoc is a read of one issue with the key echo: the reply value at /key must equal the issueKey argument. It
// names example hosts only.
const echoDoc = `version: 1
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

// TestKeyEchoValidates checks that a single-object read may declare one echo: the parameter that carries the key
// and the pointer that the reply must echo it at.
func TestKeyEchoValidates(t *testing.T) {
	if r := Validate([]byte(echoDoc)); !r.Valid() {
		t.Fatalf("a read with a key echo is refused: %v", r.Err())
	}
}

// TestKeyEchoOnListIsRefused checks that the load refuses an echo on a list response: only a read of one object
// may declare one.
func TestKeyEchoOnListIsRefused(t *testing.T) {
	doc := mutateAll(echoDoc,
		"      echo: { param: issueKey, pointer: /key }\n      fields:",
		"      echo: { param: issueKey, pointer: /key }\n      items: /issues\n      fields:")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an echo on a list response is valid")
	}
	if !hasFinding(r, "/operations/0/response/echo") {
		t.Fatalf("no finding names the echo:\n%s", allText(r))
	}
}

// TestKeyEchoNamesNoParameterIsRefused checks that the echo names a parameter of the operation.
func TestKeyEchoNamesNoParameterIsRefused(t *testing.T) {
	doc := mutateAll(echoDoc, "echo: { param: issueKey, pointer: /key }", "echo: { param: other, pointer: /key }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an echo naming no parameter is valid")
	}
	if !hasFinding(r, "/operations/0/response/echo/param") {
		t.Fatalf("no finding names the echo parameter:\n%s", allText(r))
	}
}

// TestKeyEchoWithoutPointerIsRefused checks that the echo names the pointer the reply must echo the key at.
func TestKeyEchoWithoutPointerIsRefused(t *testing.T) {
	doc := mutateAll(echoDoc, "echo: { param: issueKey, pointer: /key }", "echo: { param: issueKey, pointer: '' }")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an echo with no pointer is valid")
	}
	if !hasFinding(r, "/operations/0/response/echo/pointer") {
		t.Fatalf("no finding names the echo pointer:\n%s", allText(r))
	}
}

// TestKeyEchoOnWriteIsRefused checks that only a read declares an echo: a write of one object takes none.
func TestKeyEchoOnWriteIsRefused(t *testing.T) {
	doc := mutateAll(echoDoc, "    access: read\n    method: GET", "    access: write\n    method: POST")
	r := Validate([]byte(doc))
	if r.Valid() {
		t.Fatal("an echo on a write is valid")
	}
	if !hasFinding(r, "/operations/0/response/echo") {
		t.Fatalf("no finding names the echo:\n%s", allText(r))
	}
}

// TestReviewListsKeyEchoes checks the review block of the reads with a key echo: each names the parameter and the
// pointer the reply must echo. A spec with no echo writes no block, so the golden review keeps its text.
func TestReviewListsKeyEchoes(t *testing.T) {
	review := Review(requireValid(t, echoDoc).Spec)
	for _, line := range []string{
		"Key echoes: 1",
		"1. example-wiki.issues.get: issueKey equals /key",
	} {
		if !strings.Contains(review, line) {
			t.Fatalf("the review has no line %q:\n%s", line, review)
		}
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Key echoes") {
		t.Fatal("the review of a spec with no echo names key echoes")
	}
}
