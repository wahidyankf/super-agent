package unit_test

import (
	"fmt"
	"strings"
	"testing"

	spec "github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// pageAllOp is a read of every page with a fixed page size of 25 and a cap of 20 pages, at index zero.
const pageAllOp = `  - id: example-wiki.docs.all
    summary: List every document.
    access: read
    method: GET
    path: /docs
    pagination:
      style: page-all
      page_size: 25
      max_pages: 20
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
        expect: {}
`

// offsetOp is an offset read at index zero whose page end is the last flag and a cut to the limit parameter.
const offsetOp = `  - id: example-wiki.docs.list
    summary: List documents.
    access: read
    method: GET
    path: /docs
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
    pagination:
      style: offset
      max_pages: 5
      limit_param: limit
      offset_param: offset
      last_pointer: /isLast
      cut_to_limit: limit
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
            isLast: true
        expect: {}
`

// bodyOffsetOp is an offset read sent over POST at index four, with its limit and offset in the JSON body.
const bodyOffsetOp = `  - id: example-wiki.docs.page
    summary: Page through documents with a body offset.
    access: read
    method: POST
    read_over_post: true
    path: /docs/search
    params:
      - name: limit
        type: Limit
        in: body
        required: false
        default: 20
      - name: skip
        type: Offset
        in: body
        required: false
        default: 0
    pagination:
      style: offset
      in: body
      max_pages: 5
      limit_param: limit
      offset_param: skip
    body:
      limit: "{{limit}}"
      skip: "{{skip}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 20
          skip: 0
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect: {}
`

// bodyCursorOp is a cursor read sent over POST at index five, with its cursor in the JSON body.
const bodyCursorOp = `  - id: example-wiki.docs.walk
    summary: Walk documents with a body cursor.
    access: read
    method: POST
    read_over_post: true
    path: /docs/walk
    params:
      - name: cursor
        type: ID
        in: body
        required: false
    pagination:
      style: cursor
      in: body
      max_pages: 5
      cursor_param: cursor
      next_pointer: /next_cursor
    body:
      cursor: "{{cursor}}"
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
            next_cursor: null
        expect: {}
`

// hasFindingUnder reports whether the result holds a finding at a path that starts with prefix.
func hasFindingUnder(r spec.Result, prefix string) bool {
	for _, f := range r.Findings {
		if strings.HasPrefix(f.Path, prefix) {
			return true
		}
	}
	return false
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A page-all read with a fixed page size in range is accepted
func TestPageAllReadWithFixedPageSizeInRangeIsAccepted(t *testing.T) {
	for _, size := range []int{1, 25, 100} {
		t.Run(fmt.Sprintf("size %d", size), func(t *testing.T) {
			requireAccepted(t, mutate(t, oneOp(pageAllOp), "      page_size: 25", fmt.Sprintf("      page_size: %d", size)))
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A page-all page size outside 1 to 100 is refused at the page size
func TestPageAllPageSizeOutside1To100IsRefusedAtThePageSize(t *testing.T) {
	for _, size := range []int{0, 101} {
		t.Run(fmt.Sprintf("size %d", size), func(t *testing.T) {
			doc := mutate(t, oneOp(pageAllOp), "      page_size: 25", fmt.Sprintf("      page_size: %d", size))
			requireRefusedAt(t, doc, "/operations/0/pagination/page_size")
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A page-all read without a page size, or without a page cap, is refused
func TestPageAllReadWithoutPageSizeOrPageCapIsRefused(t *testing.T) {
	cases := []struct {
		name, old, repl, path string
	}{
		{"omits its page size", "      page_size: 25\n", "", "/operations/0/pagination/page_size"},
		{"sets its page cap to 0", "      max_pages: 20", "      max_pages: 0", "/operations/0/pagination/max_pages"},
		{"sets its page cap to 21", "      max_pages: 20", "      max_pages: 21", "/operations/0/pagination/max_pages"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireRefusedAt(t, mutate(t, oneOp(pageAllOp), c.old, c.repl), c.path)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A page-all read may send its paging values in the query or in the body
func TestPageAllReadMaySendItsPagingValuesInTheQueryOrInTheBody(t *testing.T) {
	cases := []struct {
		name, doc string
	}{
		{"query", mutate(t, oneOp(pageAllOp), "      page_size: 25\n", "      page_size: 25\n      in: query\n")},
		{"body", mutate(t, oneOp(pageAllOp), "    method: GET\n", "    method: POST\n    read_over_post: true\n",
			"    path: /docs\n", "    path: /docs\n    body: {}\n", "      page_size: 25\n", "      page_size: 25\n      in: body\n")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireAccepted(t, c.doc)
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A page-all read whose paging values travel in a header is refused at the location
func TestPageAllReadWhosePagingValuesTravelInAHeaderIsRefusedAtTheLocation(t *testing.T) {
	doc := mutate(t, oneOp(pageAllOp), "      page_size: 25\n", "      page_size: 25\n      in: header\n")
	requireRefusedAt(t, doc, "/operations/0/pagination/in")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A page-all read that names a parameter or a next pointer is refused
func TestPageAllReadNamingParameterOrNextPointerIsRefused(t *testing.T) {
	cases := []struct {
		name, repl string
	}{
		{"a limit parameter", "      limit_param: limit\n"},
		{"an offset parameter", "      offset_param: offset\n"},
		{"a cursor parameter", "      cursor_param: cursor\n"},
		{"a next-page pointer", "      next_pointer: /next\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := mutate(t, oneOp(pageAllOp), "      page_size: 25\n", "      page_size: 25\n"+c.repl)
			requireRefusedAt(t, doc, "/operations/0/pagination")
		})
	}
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A page-all read that sets a last-page pointer is refused at the pointer
func TestPageAllReadThatSetsALastPagePointerIsRefusedAtThePointer(t *testing.T) {
	doc := mutate(t, oneOp(pageAllOp), "      page_size: 25\n", "      page_size: 25\n      last_pointer: /isLast\n")
	requireRefusedAt(t, doc, "/operations/0/pagination/last_pointer")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A cursor parameter that the operation does not declare as a query parameter is refused
func TestCursorParameterNotDeclaredAsQueryParameterIsRefused(t *testing.T) {
	doc := mutate(t, exampleWiki, "      cursor_param: cursor\n", "      cursor_param: missing\n")
	requireRefusedAt(t, doc, "/operations/0/pagination/cursor_param")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A query cursor parameter that the operation declares in the body is refused
func TestQueryCursorParameterDeclaredInBodyIsRefused(t *testing.T) {
	doc := mutate(t, exampleWiki, "        in: query\n        required: false\n        help: The cursor of the next page.",
		"        in: body\n        required: false\n        help: The cursor of the next page.")
	requireRefusedAt(t, doc, "/operations/0/pagination/cursor_param")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A body-paged offset read that names a query parameter is refused at the offset
func TestBodyPagedOffsetReadNamingQueryParameterIsRefusedAtTheOffset(t *testing.T) {
	requireAccepted(t, withOps(bodyOffsetOp))
	doc := mutate(t, withOps(bodyOffsetOp), "      - name: skip\n        type: Offset\n        in: body",
		"      - name: skip\n        type: Offset\n        in: query")
	requireRefusedAt(t, doc, "/operations/4/pagination/offset_param")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A body-paged cursor read that names a query parameter is refused at the cursor
func TestBodyPagedCursorReadNamingQueryParameterIsRefusedAtTheCursor(t *testing.T) {
	requireAccepted(t, withOps(bodyOffsetOp, bodyCursorOp))
	doc := mutate(t, withOps(bodyOffsetOp, bodyCursorOp), "      - name: cursor\n        type: ID\n        in: body",
		"      - name: cursor\n        type: ID\n        in: query")
	requireRefusedAt(t, doc, "/operations/5/pagination/cursor_param")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A cut to the limit that names no parameter of the operation is refused at the cut
func TestCutToLimitNamingNoParameterIsRefusedAtTheCut(t *testing.T) {
	requireAccepted(t, oneOp(offsetOp))
	doc := mutate(t, oneOp(offsetOp), "      cut_to_limit: limit", "      cut_to_limit: missing")
	requireRefusedAt(t, doc, "/operations/0/pagination/cut_to_limit")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A truncated block on an offset read is refused at the response
func TestTruncatedBlockOnOffsetReadIsRefusedAtTheResponse(t *testing.T) {
	doc := mutate(t, oneOp(offsetOp), "    response:\n      items: /items\n",
		"    response:\n      items: /items\n      truncated:\n        token: /next\n        last: /isLast\n")
	requireRefusedAt(t, doc, "/operations/0/response/truncated")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A truncated block whose reply has no last-page pointer is refused
func TestTruncatedBlockWithNoLastPagePointerIsRefused(t *testing.T) {
	doc := mutate(t, exampleWiki, "    response:\n      items: /items\n",
		"    response:\n      items: /items\n      truncated:\n        token: /next_cursor\n        last: /isLast\n")
	requireRefused(t, doc)
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A paged read with no stop rule is refused at the page cap
func TestPagedReadWithNoStopRuleIsRefusedAtThePageCap(t *testing.T) {
	doc := mutate(t, exampleWiki, "      max_pages: 5\n      cursor_param", "      cursor_param")
	requireRefusedAt(t, doc, "/operations/0/pagination/max_pages")
}

// @scenario specs/libs/retriever-spec/behaviours/pagination.feature: A paging rule that mixes a query parameter with a body parameter is refused
func TestPagingRuleThatMixesQueryAndBodyParameterIsRefused(t *testing.T) {
	doc := mutate(t, withOps(bodyOffsetOp), "      - name: limit\n        type: Limit\n        in: body",
		"      - name: limit\n        type: Limit\n        in: query")
	requireRefused(t, doc)
}
