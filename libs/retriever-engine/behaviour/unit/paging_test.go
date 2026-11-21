package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// The paging operations of the bindings. Each one is one operation of its own document, so a test reads exactly the
// paging rule it names.

// pagingOffsetOp is an offset read with the limit and the offset in the query, a default page of two, and a page cap of
// five.
const pagingOffsetOp = `  - id: example-wiki.docs.page
    summary: List one page of documents by offset.
    access: read
    method: GET
    path: /docs/page
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
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
        expect:
          id: doc-1
`

// pagingCursorOp is a cursor read with the cursor in the query and a default page of twenty, with a page cap of five.
const pagingCursorOp = `  - id: example-wiki.docs.list
    summary: List documents by cursor.
    access: read
    method: GET
    path: /docs
    params:
      - name: cursor
        type: ID
        in: query
        required: false
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
    pagination:
      style: cursor
      max_pages: 5
      cursor_param: cursor
      next_pointer: /next_cursor
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          limit: 20
        response:
          status: 200
          body:
            items:
              - id: doc-1
            next_cursor: null
        expect:
          id: doc-1
`

// pagingPageAllOp is a page-all read of the query with a page size of two, and a page cap of five.
const pagingPageAllOp = `  - id: example-wiki.docs.all
    summary: List every document, page by page.
    access: read
    method: GET
    path: /docs
    pagination:
      style: page-all
      max_pages: 5
      page_size: 2
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

// pagingPageAllBodyOp is a page-all read sent as a POST, which declares a filter and a constant limit of 99 in its body.
// The paging keys must replace the constant, and the filter must survive.
const pagingPageAllBodyOp = `  - id: example-wiki.docs.search
    summary: Search documents by a filter, page by page.
    access: read
    method: POST
    read_over_post: true
    path: /docs/search
    params:
      - name: filter
        type: Text
        in: body
        required: false
    body:
      filter: "{{filter}}"
      limit: 99
    pagination:
      style: page-all
      in: body
      max_pages: 5
      page_size: 2
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          filter: active
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// pagingBodyOffsetOp is an offset read sent as a POST, with the limit and the offset in the body, and a page cap of 20.
const pagingBodyOffsetOp = `  - id: example-wiki.docs.search
    summary: Search documents by offset in the body.
    access: read
    method: POST
    read_over_post: true
    path: /docs/search
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
        expect:
          id: doc-1
`

// pagingBodyCursorOp is a cursor read sent as a POST, with the cursor and the limit in the body.
const pagingBodyCursorOp = `  - id: example-wiki.docs.search
    summary: Search documents by cursor in the body.
    access: read
    method: POST
    read_over_post: true
    path: /docs/search
    params:
      - name: cursor
        type: ID
        in: body
        required: false
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
      max_pages: 5
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

// pagingTruncatedOp is a cursor read with the page-end rule: a last flag, a cut to the limit, and the computed truncated
// flag of each item, which reads a next-page token.
const pagingTruncatedOp = `  - id: example-wiki.docs.list
    summary: List documents by cursor, with the page-end rule.
    access: read
    method: GET
    path: /docs
    params:
      - name: cursor
        type: ID
        in: query
        required: false
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
    pagination:
      style: cursor
      max_pages: 5
      cursor_param: cursor
      next_pointer: /next_cursor
      last_pointer: /isLast
      cut_to_limit: limit
    response:
      items: /items
      truncated:
        token: /next_token
        last: /isLast
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
            isLast: true
        expect:
          id: doc-1
`

// pagingVary returns the document with one line changed. It fails the test when the line is not in the document, so a
// variant can never silently equal its base.
func pagingVary(t *testing.T, doc, from, to string) string {
	t.Helper()
	if !strings.Contains(doc, from) {
		t.Fatalf("the document has no %q to change", from)
	}
	return strings.Replace(doc, from, to, 1)
}

// pagingPage returns a reply with the documents of one page under /items, and the extra top-level fields of the reply.
func pagingPage(ids []string, extra map[string]any) rebindReply {
	items := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, map[string]any{"id": id})
	}
	body := map[string]any{"items": items}
	for k, v := range extra {
		body[k] = v
	}
	raw, _ := json.Marshal(body)
	return rebindJSON(http.StatusOK, string(raw))
}

// pagingRead runs one read of the operation with the arguments.
func pagingRead(eng *engine.Engine, op string, args map[string]any) (domain.Record, error) {
	return eng.Read(context.Background(), domain.OperationID(op), domain.NewArgs(args))
}

// pagingDocs returns the document ids doc-(from) to doc-(to) inclusive, as the ids of one page.
func pagingDocs(from, to int) []string {
	var ids []string
	for i := from; i <= to; i++ {
		ids = append(ids, fmt.Sprintf("doc-%d", i))
	}
	return ids
}

// pagingFullPage answers request n with a full page of two documents, numbered from the request index.
func pagingFullPage(n int) rebindReply {
	return pagingPage(pagingDocs(2*n+1, 2*n+2), nil)
}

// TestPaging_OffsetCollectsEachPage binds the offset walk to its short page.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: An offset read collects each page until a short page
func TestPaging_OffsetCollectsEachPage(t *testing.T) {
	s := rebindSpec(t, pagingOffsetOp)
	tr := rebindReplies(
		pagingPage(pagingDocs(1, 2), nil),
		pagingPage(pagingDocs(3, 4), nil),
		pagingPage(pagingDocs(5, 5), nil),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := strings.Join(rebindIDs(rec), ","); got != "doc-1,doc-2,doc-3,doc-4,doc-5" {
		t.Fatalf("the items are %s, want all three pages", got)
	}
	reqs := tr.seen()
	if len(reqs) != 3 {
		t.Fatalf("%d requests, want 3", len(reqs))
	}
	for i, want := range []string{"0", "2", "4"} {
		if got := reqs[i].url.Query().Get("offset"); got != want {
			t.Fatalf("request %d offset is %q, want %q", i, got, want)
		}
	}
}

// TestPaging_OffsetStopsAtSpecPageCap binds the offset page cap of a spec to the continuation it reports.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: An offset read stops at the page cap of the spec and says that more rows exist
func TestPaging_OffsetStopsAtSpecPageCap(t *testing.T) {
	s := rebindSpec(t, pagingVary(t, pagingOffsetOp, "max_pages: 5", "max_pages: 2"))
	tr := rebindFunc(func(n int, _ rebindRequest) rebindReply { return pagingFullPage(n) })
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if n := len(tr.seen()); n != 2 {
		t.Fatalf("%d requests, want the 2 pages of the cap", n)
	}
	if got := len(rec.Items()); got != 4 {
		t.Fatalf("%d items, want the 4 items of the 2 pages", got)
	}
	page, ok := rec.Page()
	if !ok || !page.HasMore {
		t.Fatalf("the result says no more rows exist, want more rows")
	}
}

// TestPaging_CursorFollowsNextCursorUntilNull binds the cursor walk to its null next cursor.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A cursor read follows the next cursor until it is null
func TestPaging_CursorFollowsNextCursorUntilNull(t *testing.T) {
	s := rebindSpec(t, pagingCursorOp)
	tr := rebindReplies(
		pagingPage([]string{"doc-1"}, map[string]any{"next_cursor": "c2"}),
		pagingPage([]string{"doc-2"}, map[string]any{"next_cursor": nil}),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.list", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := strings.Join(rebindIDs(rec), ","); got != "doc-1,doc-2" {
		t.Fatalf("the items are %s, want the items of both pages", got)
	}
	if _, reported := rec.Page(); reported {
		t.Fatalf("a continuation is reported, want none")
	}
	reqs := tr.seen()
	if len(reqs) != 2 || reqs[1].url.Query().Get("cursor") != "c2" {
		t.Fatalf("the second request does not carry the cursor c2: %d requests", len(reqs))
	}
}

// TestPaging_CursorNeverEndingStopsAtCap binds a cursor source that never ends to the page cap.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A cursor read whose source never ends stops at the cap without a continuation
func TestPaging_CursorNeverEndingStopsAtCap(t *testing.T) {
	s := rebindSpec(t, pagingCursorOp)
	tr := rebindFunc(func(n int, _ rebindRequest) rebindReply {
		return pagingPage(pagingDocs(n+1, n+1), map[string]any{"next_cursor": fmt.Sprintf("c%d", n+1)})
	})
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.list", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if n := len(tr.seen()); n != 5 {
		t.Fatalf("%d requests, want the cap of five pages", n)
	}
	if _, reported := rec.Page(); reported {
		t.Fatalf("a continuation is reported, want none")
	}
}

// TestPaging_PagedReadFailingPageStopsWithItsError binds each failing page to the error of that page and to no later
// request.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A paged read whose <page> fails stops with the error of that page
func TestPaging_PagedReadFailingPageStopsWithItsError(t *testing.T) {
	rows := []struct {
		page, style string
		failAt      int
	}{
		{page: "first page", style: "cursor", failAt: 0},
		{page: "second page", style: "offset", failAt: 1},
	}
	for _, row := range rows {
		t.Run(row.style+" read, "+row.page+" fails", func(t *testing.T) {
			op, args := pagingOffsetOp, map[string]any{"limit": 2}
			if row.style == "cursor" {
				op, args = pagingCursorOp, nil
			}
			s := rebindSpec(t, op)
			tr := rebindFunc(func(n int, _ rebindRequest) rebindReply {
				if n == row.failAt {
					return rebindJSON(http.StatusInternalServerError, "")
				}
				if row.style == "cursor" {
					return pagingPage(pagingDocs(1, 1), map[string]any{"next_cursor": "c1"})
				}
				return pagingFullPage(n)
			})
			eng := rebindEngine(t, s, tr, &rebindClock{})
			opID := "example-wiki.docs.page"
			if row.style == "cursor" {
				opID = "example-wiki.docs.list"
			}
			_, err := pagingRead(eng, opID, args)
			if !errors.Is(err, domain.ErrUpstreamFailure) {
				t.Fatalf("the error is %v, want the error of the failed page", err)
			}
			if n := len(tr.seen()); n != row.failAt+1 {
				t.Fatalf("%d requests, want %d: no later page is requested", n, row.failAt+1)
			}
		})
	}
}

// TestPaging_NonListPageFailsRead binds a page whose items are not a list to an unexpected response, in both paged styles.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page whose items are not a list fails the read
func TestPaging_NonListPageFailsRead(t *testing.T) {
	rows := []struct {
		style, op string
		args      map[string]any
	}{
		{style: "offset", op: "example-wiki.docs.page", args: map[string]any{"limit": 2}},
		{style: "cursor", op: "example-wiki.docs.list"},
	}
	for _, row := range rows {
		t.Run(row.style, func(t *testing.T) {
			doc := pagingOffsetOp
			if row.style == "cursor" {
				doc = pagingCursorOp
			}
			s := rebindSpec(t, doc)
			tr := rebindReplies(rebindJSON(http.StatusOK, `{"items": "not a list"}`))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			_, err := pagingRead(eng, row.op, row.args)
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("the error is %v, want an unexpected response", err)
			}
		})
	}
}

// TestPaging_OffsetWithoutPageSizeRefused binds an offset read whose page size has no value to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: An offset read with no page size is refused before any request
func TestPaging_OffsetWithoutPageSizeRefused(t *testing.T) {
	doc := pagingVary(t, pagingOffsetOp, "        default: 2\n      - name: offset", "      - name: offset")
	s := rebindSpec(t, doc)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"offset": 0})
	if !errors.Is(err, domain.ErrInvalidLimit) {
		t.Fatalf("the error is %v, want the refused page size", err)
	}
	if n := len(tr.seen()); n != 0 {
		t.Fatalf("%d requests were sent, want none", n)
	}
}

// TestPaging_CursorWholeNumberIsFollowed binds a next cursor sent as a whole number to the next page.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A cursor that the source sends as a whole number is followed
func TestPaging_CursorWholeNumberIsFollowed(t *testing.T) {
	s := rebindSpec(t, pagingCursorOp)
	tr := rebindReplies(
		pagingPage([]string{"doc-1"}, map[string]any{"next_cursor": 42}),
		pagingPage([]string{"doc-2"}, nil),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.list", nil); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 2 || reqs[1].url.Query().Get("cursor") != "42" {
		t.Fatalf("the read does not follow the cursor 42 to the next page: %d requests", len(reqs))
	}
}

// TestPaging_ObjectNextCursorFailsRead binds a next cursor that is an object to an unexpected response.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A next cursor that is an object fails the read
func TestPaging_ObjectNextCursorFailsRead(t *testing.T) {
	s := rebindSpec(t, pagingCursorOp)
	tr := rebindReplies(pagingPage([]string{"doc-1"}, map[string]any{"next_cursor": map[string]any{"value": "c2"}}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.list", nil); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("the error is %v, want an unexpected response", err)
	}
}

// TestPaging_PageAllCollectsEveryPage binds the page-all walk to its short page, with no continuation.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page-all read collects every page until a short page
func TestPaging_PageAllCollectsEveryPage(t *testing.T) {
	s := rebindSpec(t, pagingPageAllOp)
	tr := rebindReplies(
		pagingPage(pagingDocs(1, 2), nil),
		pagingPage(pagingDocs(3, 4), nil),
		pagingPage(pagingDocs(5, 5), nil),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.all", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := strings.Join(rebindIDs(rec), ","); got != "doc-1,doc-2,doc-3,doc-4,doc-5" {
		t.Fatalf("the items are %s, want all three pages", got)
	}
	if _, reported := rec.Page(); reported {
		t.Fatalf("a continuation is reported, want none")
	}
}

// TestPaging_PageAllFailsAtCap binds a page-all read that keeps returning full pages to the page-cap error.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page-all read fails with the page-cap error after the third page
func TestPaging_PageAllFailsAtCap(t *testing.T) {
	s := rebindSpec(t, pagingVary(t, pagingPageAllOp, "max_pages: 5", "max_pages: 3"))
	tr := rebindFunc(func(n int, _ rebindRequest) rebindReply { return pagingFullPage(n) })
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.all", nil)
	if !errors.Is(err, domain.ErrPageCapReached) {
		t.Fatalf("the error is %v, want the page-cap error", err)
	}
	if n := len(tr.seen()); n != 3 {
		t.Fatalf("%d requests, want 3 pages before the cap error", n)
	}
}

// TestPaging_PageAllNonListPageReturnsNoItems binds a page-all page that is not a list to an unexpected response with no
// items.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page-all read whose page is not a list fails the read without mapping part of it
func TestPaging_PageAllNonListPageReturnsNoItems(t *testing.T) {
	s := rebindSpec(t, pagingPageAllOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil), rebindJSON(http.StatusOK, `{"items": 7}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.all", nil)
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("the error is %v, want an unexpected response", err)
	}
	if n := len(rec.Items()); n != 0 {
		t.Fatalf("%d items are returned, want none", n)
	}
}

// TestPaging_PageAllWithoutBodySendsPagingKeysInQuery binds the paging keys of a query page-all read to the query.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page-all read with no body location sends its paging keys in the query
func TestPaging_PageAllWithoutBodySendsPagingKeysInQuery(t *testing.T) {
	s := rebindSpec(t, pagingPageAllOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil), pagingPage(pagingDocs(3, 3), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.all", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want 2", len(reqs))
	}
	for i, want := range []string{"0", "2"} {
		q := reqs[i].url.Query()
		if q.Get("limit") != "2" || q.Get("offset") != want {
			t.Fatalf("request %d carries limit %q and offset %q, want 2 and %s", i, q.Get("limit"), q.Get("offset"), want)
		}
	}
	if got := len(rec.Items()); got != 3 {
		t.Fatalf("%d items, want the items of both pages", got)
	}
}

// TestPaging_PageAllBodyKeepsDeclaredKeys binds a page-all body to its declared filter, and to the paging keys that replace
// the declared limit.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page-all body keeps its declared keys and the paging keys win over a declared limit
func TestPaging_PageAllBodyKeepsDeclaredKeys(t *testing.T) {
	s := rebindSpec(t, pagingPageAllBodyOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 1), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.search", map[string]any{"filter": "active"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want 1", len(reqs))
	}
	var body map[string]any
	if err := json.Unmarshal(reqs[0].body, &body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}
	if body["filter"] != "active" {
		t.Fatalf("the body filter is %v, want the declared filter", body["filter"])
	}
	if body["limit"] != float64(2) || body["offset"] != float64(0) {
		t.Fatalf("the body carries limit %v and offset %v, want the paging keys 2 and 0", body["limit"], body["offset"])
	}
}

// TestPaging_PageAllPageSizeRefused binds a hand-built page-all model whose page size has no value to a refusal before any
// request.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page-all read with a page size out of range is refused before any request
func TestPaging_PageAllPageSizeRefused(t *testing.T) {
	s := rebindSpec(t, pagingPageAllOp)
	s.Operations[0].Pagination.PageSize = 0
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.all", nil); !errors.Is(err, domain.ErrInvalidLimit) {
		t.Fatalf("the error is %v, want the refused page size", err)
	}
	if n := len(tr.seen()); n != 0 {
		t.Fatalf("%d requests were sent, want none", n)
	}
}

// TestPaging_BodyOffsetStopsAtCapWithContinuation binds a body offset read to its page cap and to the continuation offset.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A body offset read stops at the page cap and reports the continuation offset
func TestPaging_BodyOffsetStopsAtCapWithContinuation(t *testing.T) {
	s := rebindSpec(t, pagingBodyOffsetOp)
	tr := rebindFunc(func(n int, _ rebindRequest) rebindReply { return pagingFullPage(n) })
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.search", map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 20 {
		t.Fatalf("%d requests, want the cap of 20 pages", len(reqs))
	}
	var last map[string]any
	if err := json.Unmarshal(reqs[len(reqs)-1].body, &last); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}
	if last["offset"] != float64(38) {
		t.Fatalf("the last page asks offset %v, want 38", last["offset"])
	}
	if got := len(rec.Items()); got != 40 {
		t.Fatalf("%d items, want the items of the first 20 pages", got)
	}
	page, ok := rec.Page()
	if !ok || page.NextOffset != 40 {
		t.Fatalf("the continuation names offset %d, want 40", page.NextOffset)
	}
}

// TestPaging_BodyCursorEchoStopsAfterSecondPage binds a body cursor that the source echoes to a stop after the second page.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A body cursor that echoes its own cursor stops after the second page
func TestPaging_BodyCursorEchoStopsAfterSecondPage(t *testing.T) {
	s := rebindSpec(t, pagingBodyCursorOp)
	tr := rebindReplies(
		pagingPage([]string{"doc-1"}, map[string]any{"next_cursor": "c1"}),
		pagingPage([]string{"doc-2"}, map[string]any{"next_cursor": "c1"}),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.search", nil); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if n := len(tr.seen()); n != 2 {
		t.Fatalf("%d requests, want the read to stop after the second page", n)
	}
}

// TestPaging_BodyCursorSendsSecondCursorInBody binds the second request of a body cursor read to the cursor in its body.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A body cursor read collects two pages and sends the second cursor in the body
func TestPaging_BodyCursorSendsSecondCursorInBody(t *testing.T) {
	s := rebindSpec(t, pagingBodyCursorOp)
	tr := rebindReplies(
		pagingPage([]string{"doc-1"}, map[string]any{"next_cursor": "c1"}),
		pagingPage([]string{"doc-2"}, map[string]any{"next_cursor": nil}),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.search", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want 2", len(reqs))
	}
	var second map[string]any
	if err := json.Unmarshal(reqs[1].body, &second); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}
	if second["cursor"] != "c1" {
		t.Fatalf("the second body carries cursor %v, want c1", second["cursor"])
	}
	if _, reported := rec.Page(); reported || len(rec.Items()) != 2 {
		t.Fatalf("the read reports a continuation or holds %d items, want none and 2", len(rec.Items()))
	}
}

// TestPaging_BodyOffsetAtIntegerCapRefused binds a body offset at the integer cap to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A body offset at or above the integer cap is refused before any request
func TestPaging_BodyOffsetAtIntegerCapRefused(t *testing.T) {
	s := rebindSpec(t, pagingBodyOffsetOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.search", map[string]any{"limit": 2, "offset": 2147483648})
	if !errors.Is(err, domain.ErrInvalidOffset) {
		t.Fatalf("the error is %v, want the refused offset", err)
	}
	if n := len(tr.seen()); n != 0 {
		t.Fatalf("%d requests were sent, want none", n)
	}
}

// TestPaging_FullLastPageClaimsNoMore binds a full page whose last flag is true to a read of one request with no
// continuation.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A full page with the last flag set claims no further page
func TestPaging_FullLastPageClaimsNoMore(t *testing.T) {
	doc := pagingVary(t, pagingOffsetOp, "      offset_param: offset\n", "      offset_param: offset\n      last_pointer: /isLast\n      cut_to_limit: limit\n")
	s := rebindSpec(t, doc)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), map[string]any{"isLast": true}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if n := len(tr.seen()); n != 1 {
		t.Fatalf("%d requests, want one", n)
	}
	if page, ok := rec.Page(); !ok || page.HasMore {
		t.Fatalf("the read claims a further page, want none")
	}
}

// TestPaging_NoLastFlagFailsNamingPointer binds a page with no last flag to an unexpected response that names the path.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page with no last flag fails the read naming the pointer path
func TestPaging_NoLastFlagFailsNamingPointer(t *testing.T) {
	doc := pagingVary(t, pagingOffsetOp, "      offset_param: offset\n", "      offset_param: offset\n      last_pointer: /isLast\n      cut_to_limit: limit\n")
	s := rebindSpec(t, doc)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"limit": 2})
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "/isLast") {
		t.Fatalf("the error is %v, want an unexpected response naming /isLast", err)
	}
}

// TestPaging_NonBoolLastFlagFailsOffsetRead binds a last flag that is not a bool to an unexpected response that names the
// path.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A last flag that is not a bool fails an offset read naming the pointer path
func TestPaging_NonBoolLastFlagFailsOffsetRead(t *testing.T) {
	doc := pagingVary(t, pagingOffsetOp, "      offset_param: offset\n", "      offset_param: offset\n      last_pointer: /isLast\n      cut_to_limit: limit\n")
	s := rebindSpec(t, doc)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), map[string]any{"isLast": "yes"}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"limit": 2})
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "/isLast") {
		t.Fatalf("the error is %v, want an unexpected response naming /isLast", err)
	}
}

// TestPaging_OverLimitPageKeepsFirstItems binds a page of more items than the cut to its first items and to a claim of a
// further page.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A page holding more items than the limit keeps the first ones and claims more
func TestPaging_OverLimitPageKeepsFirstItems(t *testing.T) {
	doc := pagingVary(t, pagingOffsetOp, "      offset_param: offset\n", "      offset_param: offset\n      last_pointer: /isLast\n      cut_to_limit: limit\n")
	doc = pagingVary(t, doc, "max_pages: 5", "max_pages: 1")
	s := rebindSpec(t, doc)
	tr := rebindReplies(pagingPage(pagingDocs(1, 3), map[string]any{"isLast": false}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := strings.Join(rebindIDs(rec), ","); got != "doc-1,doc-2" {
		t.Fatalf("the items are %s, want the first two", got)
	}
	if page, ok := rec.Page(); !ok || !page.HasMore {
		t.Fatalf("the read does not claim a further page")
	}
}

// TestPaging_CursorOverLimitPageMarksEachItemTruncated binds a cursor page over the limit to its first items, each marked
// truncated.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A cursor page over the limit keeps the first items and marks each item truncated
func TestPaging_CursorOverLimitPageMarksEachItemTruncated(t *testing.T) {
	s := rebindSpec(t, pagingTruncatedOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 3), map[string]any{"isLast": false}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.list", map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := strings.Join(rebindIDs(rec), ","); got != "doc-1,doc-2" {
		t.Fatalf("the items are %s, want the first two", got)
	}
	for i, item := range rec.Items() {
		if v, _ := item.Get("truncated"); v != true {
			t.Fatalf("item %d is not marked truncated: %v", i, v)
		}
	}
}

// TestPaging_ItemWithTokenAndNoLastIsTruncated binds an item on a page with a next token and no last flag to a truncated
// item.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: An item on a page with a token and no last flag is marked truncated
func TestPaging_ItemWithTokenAndNoLastIsTruncated(t *testing.T) {
	s := rebindSpec(t, pagingTruncatedOp)
	tr := rebindReplies(pagingPage([]string{"doc-1"}, map[string]any{"next_token": "tok-2", "isLast": false}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.list", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if len(rec.Items()) != 1 {
		t.Fatalf("%d items, want 1", len(rec.Items()))
	}
	if v, _ := rec.Items()[0].Get("truncated"); v != true {
		t.Fatalf("the item is not marked truncated: %v", v)
	}
}

// TestPaging_ItemWithoutTokenIsNotTruncated binds an item on a page with no next token to an item that is not truncated.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: An item on a page with no token is not marked truncated
func TestPaging_ItemWithoutTokenIsNotTruncated(t *testing.T) {
	s := rebindSpec(t, pagingTruncatedOp)
	tr := rebindReplies(pagingPage([]string{"doc-1"}, map[string]any{"isLast": false}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.list", nil)
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if len(rec.Items()) != 1 {
		t.Fatalf("%d items, want 1", len(rec.Items()))
	}
	if v, _ := rec.Items()[0].Get("truncated"); v != false {
		t.Fatalf("the item is marked truncated: %v", v)
	}
}

// TestPaging_WrongShapeTokenOrLastFailsNamingPath binds a token that is not text, and a last flag that is not a bool, to
// an unexpected response that names the pointer path.
// @scenario specs/libs/retriever-engine/behaviours/paging.feature: A token that is not text or a last flag that is not a bool fails the read naming the pointer path
func TestPaging_WrongShapeTokenOrLastFailsNamingPath(t *testing.T) {
	rows := []struct {
		name, path string
		extra      map[string]any
	}{
		{name: "token is a number", path: "/next_token", extra: map[string]any{"next_token": 5, "isLast": false}},
		{name: "last flag is text", path: "/isLast", extra: map[string]any{"next_token": "tok-2", "isLast": "no"}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			s := rebindSpec(t, pagingTruncatedOp)
			tr := rebindReplies(pagingPage([]string{"doc-1"}, row.extra))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			_, err := pagingRead(eng, "example-wiki.docs.list", nil)
			if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), row.path) {
				t.Fatalf("the error is %v, want an unexpected response naming %s", err, row.path)
			}
		})
	}
}
