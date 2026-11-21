package unit_test

import (
	"fmt"
	"strings"
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// postReadOp is a read sent over POST with its read-over-POST marker and an empty body.
const postReadOp = `  - id: example-wiki.docs.search
    summary: Search documents.
    access: read
    method: POST
    read_over_post: true
    path: /search
    body: {}
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            id: doc-1
        expect:
          id: doc-1
`

// callOpTemplate is one operation of the given label and method. The marker and the body lines are optional.
const callOpTemplate = `  - id: example-wiki.docs.call
    summary: Call the operation.
    access: %s
    method: %s
%s    path: /docs
%s    response:
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            id: doc-1
        expect:
          id: doc-1
`

// singleOpDoc returns a document of one operation with the label, the method, and the POST marker given. A marked
// operation carries the empty body that a POST read needs.
func singleOpDoc(access, method string, marker bool) string {
	extra, body := "", ""
	if marker {
		extra = "    read_over_post: true\n"
		body = "    body: {}\n"
	}
	return oneOp(fmt.Sprintf(callOpTemplate, access, method, extra, body))
}

// nestedBody returns a flow mapping that nests the given number of levels, with the top-level object as level one.
func nestedBody(levels int) string {
	return strings.Repeat("{k: ", levels) + "1" + strings.Repeat("}", levels)
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: The access matrix accepts a label, method, and marker pair only when it allows the pair
func TestAccessMatrixAcceptsAPairOnlyWhenItAllowsIt(t *testing.T) {
	cases := []struct {
		access, method, marker, verdict string
	}{
		{"read", "GET", "no", "accepted"},
		{"read", "HEAD", "no", "accepted"},
		{"read", "POST", "yes", "accepted"},
		{"read", "POST", "no", "refused"},
		{"read", "PUT", "no", "refused"},
		{"read", "GET", "yes", "refused"},
		{"write", "POST", "no", "accepted"},
		{"write", "PUT", "no", "accepted"},
		{"write", "PATCH", "no", "accepted"},
		{"write", "DELETE", "no", "accepted"},
		{"admin", "DELETE", "no", "accepted"},
		{"admin", "GET", "no", "refused"},
		{"owner", "GET", "no", "refused"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s %s marker %s", c.access, c.method, c.marker), func(t *testing.T) {
			marker := c.marker == "yes"
			allowed := spec.AccessMethodAllowed(c.access, c.method, marker)
			if allowed != (c.verdict == "accepted") {
				t.Fatalf("the matrix answers %v; want %s", allowed, c.verdict)
			}
			r := validate(singleOpDoc(c.access, c.method, marker))
			if r.Valid() != (c.verdict == "accepted") {
				t.Fatalf("the spec is valid = %v; want %s. findings:\n%s", r.Valid(), c.verdict, text(r))
			}
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: An access label that the spec does not define is refused at the label
func TestAccessLabelThatTheSpecDoesNotDefineIsRefusedAtTheLabel(t *testing.T) {
	doc := mutate(t, exampleWiki, "    access: read\n    method: GET\n    path: /docs\n    params:",
		"    access: owner\n    method: GET\n    path: /docs\n    params:")
	requireRefusedAt(t, doc, "/operations/0/access")
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A delete that the spec labels as a read is refused at the method
func TestDeleteLabelledAsReadIsRefusedAtTheMethod(t *testing.T) {
	doc := mutate(t, exampleWiki, "    access: write\n    method: DELETE", "    access: read\n    method: DELETE")
	requireRefusedAt(t, doc, "/operations/3/method")
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A POST read without the read-over-POST marker is refused at the operation
func TestPostReadWithoutMarkerIsRefusedAtTheOperation(t *testing.T) {
	requireAccepted(t, withOps(postReadOp))
	doc := mutate(t, withOps(postReadOp), "    read_over_post: true\n", "")
	requireRefusedAt(t, doc, "/operations/4/read_over_post")
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: The read-over-POST marker on a GET operation is refused at the marker
func TestReadOverPostMarkerOnGetIsRefusedAtTheMarker(t *testing.T) {
	doc := mutate(t, exampleWiki, "    method: GET\n    path: /docs\n", "    method: GET\n    read_over_post: true\n    path: /docs\n")
	requireRefusedAt(t, doc, "/operations/0/read_over_post")
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A POST read with no body is refused at the body
func TestPostReadWithNoBodyIsRefusedAtTheBody(t *testing.T) {
	doc := mutate(t, withOps(postReadOp), "    body: {}\n", "")
	requireRefusedAt(t, doc, "/operations/4/body")
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A POST read whose body slot names a path parameter is refused at the slot
func TestPostReadBodySlotNamingPathParameterIsRefusedAtTheSlot(t *testing.T) {
	doc := mutate(t, withOps(postReadOp), "    path: /search\n    body: {}\n",
		"    path: /docs/{id}\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n"+
			"    body:\n      collectionId: \"{{id}}\"\n")
	requireRefusedAt(t, doc, "/operations/4/body/collectionId")
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A POST read body that holds a slot inside a longer text is refused
func TestPostReadBodyWithSlotInsideLongerTextIsRefused(t *testing.T) {
	doc := mutate(t, withOps(postReadOp), "    body: {}\n",
		"    params:\n      - name: title\n        type: Name\n        in: body\n        required: true\n"+
			"    body:\n      title: \"id-{{title}}\"\n")
	requireRefused(t, doc)
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A POST read body is accepted up to eight levels of nesting and refused past eight
func TestPostReadBodyIsAcceptedUpToEightLevelsAndRefusedPastEight(t *testing.T) {
	cases := []struct {
		levels  int
		verdict string
	}{
		{8, "accepted"},
		{9, "refused"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d levels", c.levels), func(t *testing.T) {
			doc := mutate(t, withOps(postReadOp), "    body: {}\n", "    body: "+nestedBody(c.levels)+"\n")
			r := validate(doc)
			if r.Valid() != (c.verdict == "accepted") {
				t.Fatalf("the spec is valid = %v; want %s. findings:\n%s", r.Valid(), c.verdict, text(r))
			}
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A POST read may not set its own content type, in any letter case
func TestPostReadMayNotSetItsOwnContentTypeInAnyLetterCase(t *testing.T) {
	cases := []struct {
		header, path string
	}{
		{"Content-Type", "/operations/4/headers/Content-Type"},
		{"content-type", "/operations/4/headers/content-type"},
	}
	for _, c := range cases {
		t.Run(c.header, func(t *testing.T) {
			doc := mutate(t, withOps(postReadOp), "    body: {}\n",
				"    body: {}\n    headers:\n      "+c.header+": text/plain\n")
			requireRefusedAt(t, doc, c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A POST read that names a resolve on its parameter is refused at the parameter
func TestPostReadNamingResolveOnItsParameterIsRefusedAtTheParameter(t *testing.T) {
	doc := mutate(t, withOps(postReadOp), "    body: {}\n",
		"    params:\n      - name: query\n        type: Text\n        in: body\n        required: false\n"+
			"        resolve:\n          operation: example-wiki.docs.list\n          field: id\n"+
			"    body:\n      query: \"{{query}}\"\n")
	requireRefusedAt(t, doc, "/operations/4/params/0/resolve")
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: The limits of a POST read body are 64 KiB and eight levels of nesting
func TestLimitsOfPostReadBodyAre64KiBAndEightLevels(t *testing.T) {
	if spec.MaxPostBodyBytes != 65536 {
		t.Fatalf("size limit = %d bytes; want 65536", spec.MaxPostBodyBytes)
	}
	if spec.MaxPostBodyDepth != 8 {
		t.Fatalf("depth limit = %d levels; want 8", spec.MaxPostBodyDepth)
	}
}

// @scenario specs/libs/retriever-spec/behaviours/access-and-post-reads.feature: A list in a body adds one level of depth, as an object does
func TestListInBodyAddsOneLevelOfDepthAsAnObjectDoes(t *testing.T) {
	body := []any{map[string]any{"field": []any{"text"}}}
	if got := spec.BodyDepth(body); got != 3 {
		t.Fatalf("depth = %d levels; want 3", got)
	}
}
