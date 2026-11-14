package spec

import (
	"strings"
	"testing"
)

// postReadOp is a read over POST, with the marker, a body of one declared slot, and no resolve. The fault rows of E1
// append it after the delete operation of the example, so every path of the rows above keeps its index.
const postReadOp = `  - id: example-wiki.docs.find
    summary: Find documents in a collection.
    access: read
    method: POST
    read_over_post: true
    path: /docs/find
    params:
      - name: collection_id
        type: ID
        in: body
        required: false
    body:
      collectionId: "{{collection_id}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          collection_id: col-1
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// pfx returns the POST read example with the given (old, repl) pairs applied, for a fault row.
func pfx(pairs ...string) func() string {
	return func() string { return postFixtureWith(pairs...) }
}

// postFixtureWith returns the example with the POST read appended as operation 4, with the (old, repl) pairs applied to
// the whole document after the append.
func postFixtureWith(pairs ...string) string {
	all := append([]string{"        expect: {}", "        expect: {}\n" + postReadOp}, pairs...)
	return mutateAll(exampleWiki, all...)
}

// minimalOpDoc returns the minimal spec with its one operation given the label and the method, with the marker when the
// marker is true, and with a body, so that a POST read has what it needs. Only the label, the method, and the marker vary.
func minimalOpDoc(access, method string, marker bool) string {
	extra := ""
	if marker {
		extra = "    read_over_post: true\n"
	}
	return minimalWith("    access: read\n    method: GET\n",
		"    access: "+access+"\n    method: "+method+"\n"+extra+"    body: {}\n")
}

// TestAccessMethodMatrix checks every pair of label, method, and marker against the matrix of ADR 002. The matrix allows a
// GET or HEAD read, a POST read only with the marker, a POST, PUT, PATCH, or DELETE write or admin without the marker, and
// nothing else. The Go rule and the JSON Schema give the same verdict on each pair.
func TestAccessMethodMatrix(t *testing.T) {
	allowed := map[string]bool{
		"read GET false": true, "read HEAD false": true, "read POST true": true,
		"write POST false": true, "write PUT false": true, "write PATCH false": true, "write DELETE false": true,
		"admin POST false": true, "admin PUT false": true, "admin PATCH false": true, "admin DELETE false": true,
	}
	for _, access := range []string{"read", "write", "admin", "owner"} {
		for _, method := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
			for _, marker := range []bool{false, true} {
				key := access + " " + method + " " + boolWord(marker)
				want := allowed[key]
				t.Run(key, func(t *testing.T) {
					if got := AccessMethodAllowed(access, method, marker); got != want {
						t.Fatalf("AccessMethodAllowed = %v; want %v", got, want)
					}
					r := Validate([]byte(minimalOpDoc(access, method, marker)))
					if r.Valid() != want {
						t.Fatalf("the document is valid = %v; want %v:\n%s", r.Valid(), want, allText(r))
					}
				})
			}
		}
	}
}

// boolWord names a marker in the test keys.
func boolWord(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestPostReadNeedsMarker checks that a POST read is a read only with the marker, and that the marker belongs to no other
// operation. Each case is refused at the field that carries the mistake.
func TestPostReadNeedsMarker(t *testing.T) {
	requireValid(t, postFixtureWith())
	requireRefused(t, []byte(postFixtureWith("    read_over_post: true\n", "")), "/operations/4/read_over_post")
	requireRefused(t, []byte(minimalWith("    access: read\n    method: GET\n", "    access: read\n    method: GET\n    read_over_post: true\n")),
		"/operations/0/read_over_post")
	requireRefused(t, []byte(postFixtureWith("    read_over_post: true\n", "    read_over_post: false\n")), "/operations/4/read_over_post")
	if rows := rowsWithThreat("L1"); len(rows) < 4 {
		t.Fatalf("the rows that prove L1 are %d; want the marker cases", len(rows))
	}
	runFaults(t, rowsWithThreat("L1"))
}

// TestPostReadBodyOnlyFromDeclaredSlots checks the body rules of a POST read at the spec level: every value is a constant
// or a whole slot of a declared body parameter, and no slot names another kind of parameter. The engine test sends a value
// that looks like a slot and finds it literal.
func TestPostReadBodyOnlyFromDeclaredSlots(t *testing.T) {
	requireValid(t, postFixtureWith())
	runFaults(t, rowsWithThreat("L2"))
}

// TestPostReadContentTypeFixed checks that a spec may not set the content type of a POST read, in any letter case. The
// engine sets application/json itself.
func TestPostReadContentTypeFixed(t *testing.T) {
	requireRefused(t, []byte(postFixtureWith(`      collectionId: "{{collection_id}}"`+"\n    response:",
		`      collectionId: "{{collection_id}}"`+"\n    headers:\n      Content-Type: text/plain\n    response:")),
		"/operations/4/headers/Content-Type")
	requireRefused(t, []byte(postFixtureWith(`      collectionId: "{{collection_id}}"`+"\n    response:",
		`      collectionId: "{{collection_id}}"`+"\n    headers:\n      content-type: text/plain\n    response:")),
		"/operations/4/headers/content-type")
}

// TestPostReadHasNoResolve checks that a POST read claims no side effect, so no parameter of it may resolve a name.
func TestPostReadHasNoResolve(t *testing.T) {
	requireRefused(t, []byte(postFixtureWith("      - name: collection_id\n        type: ID\n        in: body\n        required: false\n",
		"      - name: collection_id\n        type: ID\n        in: body\n        required: false\n        resolve:\n          operation: example-wiki.docs.list\n          field: id\n")),
		"/operations/4/params/0/resolve")
}

// TestPostReadBodyLimitsOfNesting checks the depth of a POST read body: a body that nests eight levels passes, and one that
// nests nine is refused.
func TestPostReadBodyLimitsOfNesting(t *testing.T) {
	eight := `{a: {b: {c: {d: {e: {f: {g: 1}}}}}}}`
	requireValid(t, postFixtureWith(`      collectionId: "{{collection_id}}"`, "      collectionId: "+eight))
	nine := `{a: {b: {c: {d: {e: {f: {g: {h: 1}}}}}}}}`
	requireRefused(t, []byte(postFixtureWith(`      collectionId: "{{collection_id}}"`, "      collectionId: "+nine)), "/operations/4/body")
}

// TestReviewListsPostReads checks the review of a POST read: its own block names the operation, its method and path, and
// the slots its body takes. A spec with no POST read has no such block, so its review keeps its text.
func TestReviewListsPostReads(t *testing.T) {
	review := Review(requireValid(t, postFixtureWith()).Spec)
	block := "POST reads: 1\n1. example-wiki.docs.find: POST /v1/docs/find, body from slots: collection_id\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no POST read block:\n%s", review)
	}
	if !strings.Contains(review, "5. example-wiki.docs.find: read, POST /v1/docs/find, writes no") {
		t.Fatalf("the operation line does not name the POST read:\n%s", review)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "POST reads") {
		t.Fatal("a spec with no POST read has a POST read block")
	}
}

// TestCheckAccessMatrixNamesEachBrokenRule checks the Go findings of a model built by hand, which the engine constructor
// meets as well. A read label with a write method is refused at its method path, and a POST read with no body at its body.
func TestCheckAccessMatrixNamesEachBrokenRule(t *testing.T) {
	admin := &Spec{Operations: []Operation{{Access: "admin", Method: "GET"}}}
	if fs := CheckAccessMatrix(admin); len(fs) != 1 || fs[0].Path != "/operations/0/method" || fs[0].Rule != ruleMatrix {
		t.Fatalf("an admin label over GET: findings %+v", fs)
	}
	noBody := &Spec{Operations: []Operation{{Access: "read", Method: "POST", ReadOverPost: true}}}
	if fs := CheckAccessMatrix(noBody); len(fs) != 1 || fs[0].Path != "/operations/0/body" || fs[0].Rule != ruleBodyMissing {
		t.Fatalf("a POST read with no body: findings %+v", fs)
	}
}

// TestBodyDepthCountsLists checks that a list adds a level as an object does, so a body cannot nest past the limit
// through its lists.
func TestBodyDepthCountsLists(t *testing.T) {
	if got := BodyDepth([]any{map[string]any{"a": []any{"x"}}}); got != 3 {
		t.Fatalf("depth %d; want 3", got)
	}
}

// TestBodySlotNamesReachEveryList checks that a slot inside a list of the body is named, so the slot rule sees it.
func TestBodySlotNamesReachEveryList(t *testing.T) {
	out := map[string]bool{}
	bodySlotNames([]any{map[string]any{"ids": []any{"{{doc_id}}"}}}, out)
	if !out["doc_id"] || len(out) != 1 {
		t.Fatalf("slots %v; want doc_id only", out)
	}
}
