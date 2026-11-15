package spec

import "testing"

// TestEditorTextConverterValidates checks that a reply field may name the editor-data text converter: the engine
// reads the editor JSON of the pointer as plain text, as the Go comment projection does. The converter is a pure
// function of the reply: it sends no request and reaches no other host.
func TestEditorTextConverterValidates(t *testing.T) {
	doc := `version: 1
kind: example-wiki
summary: Reads a comment as plain text.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.comments.get
    summary: Read one comment.
    access: read
    method: POST
    read_over_post: true
    path: /comments
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
          pointer: /data/id
          required: true
        text:
          pointer: /data/data
          fallback_pointer: /data/text
          convert: text-from-editor-data
    examples:
      - request:
          id: c1
        response:
          status: 200
          body:
            data:
              id: c1
              text: Nice
        expect:
          id: c1
`
	if res := Validate([]byte(doc)); !res.Valid() {
		t.Fatalf("a field with the editor-data converter is refused: %v", res.Err())
	}
}
