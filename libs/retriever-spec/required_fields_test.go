package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

// requiredFieldOp is a read whose reply fields use the E3 keywords: a required ID checked as an ID, a UUID-checked
// field, and a share-ID-checked field. It is appended to the example as operation 4, so the paths of the rows above
// keep their indexes, as the E1 postReadOp and the E2 body paging ops do.
const requiredFieldOp = `  - id: example-wiki.docs.entry
    summary: Read one entry.
    access: read
    method: GET
    path: /entries/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        id:
          pointer: /id
          required: true
          format: id
        uid:
          pointer: /uid
          format: uuid
        slug:
          pointer: /slug
          format: share-id
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            id: doc-1
        expect:
          id: doc-1
`

// The example body carries no UUID: the R2 example guard used to refuse every 36-character UUID as a token-shaped
// run, so the engine tests serve UUID replies through scripts instead. PR21 exempts canonical UUIDs, and the bundled
// specs carry UUID examples for the conformance replay.

// efx returns the example with the required-fields read appended as operation 4, with the given (old, repl) pairs
// applied, for a fault row.
func efx(pairs ...string) func() string {
	return func() string {
		return mutateAll(mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+requiredFieldOp), pairs...)
	}
}

// TestFormatNamesAreClosed checks the closed format names of a reply field: id, uuid, and share-id load, and any other
// name is refused at its format field. A free pattern would be untrusted input, so the language takes a name only.
func TestFormatNamesAreClosed(t *testing.T) {
	r := requireValid(t, mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+requiredFieldOp))
	raw, err := json.Marshal(r.Spec.Operations[4].Response.Fields)
	if err != nil {
		t.Fatalf("the model does not encode: %v", err)
	}
	for _, want := range []string{`"required":true`, `"format":"id"`, `"format":"uuid"`, `"format":"share-id"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("the model drops %s: %s", want, raw)
		}
	}
	runFaults(t, rowsWithThreat("L4"))
}

// TestReviewListsRequiredFields checks the review of a read with required and format-checked reply fields: its own
// block names the operation, each required field, and each checked field with its closed format name. A spec with no
// required or checked field has no such block, so its review keeps its text.
func TestReviewListsRequiredFields(t *testing.T) {
	review := Review(requireValid(t, mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+requiredFieldOp)).Spec)
	block := "Required fields: 1\n1. example-wiki.docs.entry: required id; formats id=id, slug=share-id, uid=uuid\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no required-fields block:\n%s", review)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Required fields") {
		t.Fatal("a spec with no required or checked field has a required-fields block")
	}
}
