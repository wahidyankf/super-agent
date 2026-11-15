package spec

import (
	"strings"
	"testing"
)

// bodyOffsetOp is a read over POST that pages with offset and limit in the JSON body. It is appended to the example as
// operation 4, so the paths of the rows above keep their indexes, as the E1 postReadOp does.
const bodyOffsetOp = `  - id: example-wiki.docs.page
    summary: List one page of documents by offset in the body.
    access: read
    method: POST
    read_over_post: true
    path: /docs/page
    params:
      - name: limit
        type: Limit
        in: body
        required: false
        default: 2
      - name: offset
        type: Offset
        in: body
        required: false
        default: 0
    body:
      limit: "{{limit}}"
      offset: "{{offset}}"
    pagination:
      style: offset
      in: body
      max_pages: 20
      limit_param: limit
      offset_param: offset
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            items:
              - id: doc-1
              - id: doc-2
        expect:
          id: doc-1
`

// bodyCursorOp is a read over POST that pages with a cursor in the JSON body. It is appended after bodyOffsetOp as
// operation 5.
const bodyCursorOp = `  - id: example-wiki.docs.changes
    summary: List changed documents by a cursor in the body.
    access: read
    method: POST
    read_over_post: true
    path: /docs/changes
    params:
      - name: cursor
        type: ID
        in: body
        required: false
        help: The body cursor of the next page.
      - name: limit
        type: Limit
        in: body
        required: false
        default: 2
    body:
      cursor: "{{cursor}}"
      limit: "{{limit}}"
    pagination:
      style: cursor
      in: body
      max_pages: 20
      cursor_param: cursor
      next_pointer: /next_cursor
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 2
        response:
          status: 200
          body:
            items:
              - id: doc-1
            next_cursor: null
        expect:
          id: doc-1
`

// bodyPagingDoc returns the example with the body offset read and the body cursor read appended, as operations 4
// and 5.
func bodyPagingDoc() string {
	return mutateAll(exampleWiki, "        expect: {}", "        expect: {}\n"+bodyOffsetOp+bodyCursorOp)
}

// bfx returns the body paging document with the given (old, repl) pairs applied, for a fault row.
func bfx(pairs ...string) func() string {
	return func() string { return mutateAll(bodyPagingDoc(), pairs...) }
}

// TestPagingRuleRefusesMixedLocations checks the location rule of a paging rule: with in: body every named parameter
// is a declared body parameter of the same operation, and with the default in: query every named parameter is a query
// parameter. A rule cannot mix locations. Both body styles load, and each mixed rule is refused at its paging field.
func TestPagingRuleRefusesMixedLocations(t *testing.T) {
	requireValid(t, bodyPagingDoc())
	if rows := rowsWithThreat("L3"); len(rows) < 4 {
		t.Fatalf("the rows that prove L3 are %d; want at least the mixed-location cases", len(rows))
	}
	runFaults(t, rowsWithThreat("L3"))
}

// TestPagingLocationNamesQueryOrBody checks the Go finding for a hand-built model whose paging location is neither
// query nor body. A document never reaches this rule, because the schema refuses the location first.
func TestPagingLocationNamesQueryOrBody(t *testing.T) {
	op := Operation{Pagination: &Pagination{Style: "offset", In: "header", MaxPages: 5}}
	fs := checkPagination(op, "/operations/0", map[string]Param{})
	if len(fs) != 1 || fs[0].Path != "/operations/0/pagination/in" || fs[0].Rule != "is neither query nor body" {
		t.Fatalf("findings = %+v; want the location finding at the in field", fs)
	}
}

// TestReviewListsBodyPaging checks the review of a body-paged read: its own block names the operation, its style, the
// body parameters it pages by, and its page cap. A spec with no body paging has no such block, so its review keeps its
// text.
func TestReviewListsBodyPaging(t *testing.T) {
	review := Review(requireValid(t, bodyPagingDoc()).Spec)
	block := "Body paging: 2\n1. example-wiki.docs.page: offset in body (limit, offset), at most 20 pages\n" +
		"2. example-wiki.docs.changes: cursor in body (cursor), at most 20 pages\n"
	if !strings.Contains(review, block) {
		t.Fatalf("the review has no body paging block:\n%s", review)
	}
	if strings.Contains(Review(requireValid(t, exampleWiki).Spec), "Body paging") {
		t.Fatal("a spec with no body paging has a body paging block")
	}
}
