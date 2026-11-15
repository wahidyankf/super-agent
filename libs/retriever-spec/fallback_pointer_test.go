package spec

import "testing"

// TestFallbackPointerValidates checks that a reply field may name a fallback pointer: the engine reads the first
// pointer that finds a value, so one field maps the wrapped and the unwrapped reply shapes of a read. The fallback
// is one closed pointer, tried in declaration order, never a pattern.
func TestFallbackPointerValidates(t *testing.T) {
	doc := `version: 1
kind: example-wiki
summary: Reads one document in either reply shape.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.get
    summary: Read one document.
    access: read
    method: POST
    read_over_post: true
    path: /docs
    params:
      - name: id
        type: ID
        in: body
        required: true
    body:
      id: "{{id}}"
    response:
      fields:
        id:
          pointer: /data/document/id
          fallback_pointer: /data/id
          required: true
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            data:
              id: doc-1
        expect:
          id: doc-1
`
	if res := Validate([]byte(doc)); !res.Valid() {
		t.Fatalf("a field with a fallback pointer is refused: %v", res.Err())
	}
}

// TestFallbackPointerNeedsAPointer checks the Go rule behind the schema: a fallback pointer with no pointer to fall
// back from is refused at its own path, so a model built by hand meets the same refusal as a document.
func TestFallbackPointerNeedsAPointer(t *testing.T) {
	op := Operation{
		ID: "example-wiki.docs.get",
		Response: Response{Fields: map[string]Field{
			"id": {FallbackPointer: "/data/id", Required: true},
		}},
	}
	fs := CheckObjectFields(&Spec{Kind: "example-wiki", Operations: []Operation{op}})
	if len(fs) != 1 || fs[0].Path != "/operations/0/response/fields/id/fallback_pointer" {
		t.Fatalf("findings = %+v, want the fallback-pointer path", fs)
	}
	if fs[0].Rule != ruleFallbackPointer {
		t.Fatalf("finding rule = %q, want the fixed rule", fs[0].Rule)
	}
}
