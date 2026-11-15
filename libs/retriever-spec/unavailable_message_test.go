package spec

import "testing"

// TestUnavailableMessageValidates checks that a response may name the exact body message that proves the method is
// missing: a 404 carrying exactly this message fails as operation_unavailable instead of the mapped reason, so one
// status tells a missing object from a missing method. The match is equality with the declared constant only: no
// pattern, and no substring match on arbitrary text.
func TestUnavailableMessageValidates(t *testing.T) {
	doc := `version: 1
kind: example-wiki
summary: Reads one document with a missing-method rule.
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
          pointer: /data/id
          required: true
      errors:
        "404": not_found
      unavailable_message: Endpoint not found
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
		t.Fatalf("a response with an unavailable message is refused: %v", res.Err())
	}
}
