package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// bodyOffsetYAML is a read over POST that pages with offset and limit in the JSON body. It names example hosts only.
const bodyOffsetYAML = `version: 1
kind: example-wiki
summary: Reads one page of documents by offset in the body.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 20
operations:
  - id: example-wiki.docs.page
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

// bodyCursorYAML is a read over POST that pages with a cursor in the JSON body. It names example hosts only.
const bodyCursorYAML = `version: 1
kind: example-wiki
summary: Reads changed documents by a cursor in the body.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 20
  max_pages: 20
operations:
  - id: example-wiki.docs.changes
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
      - name: limit
        type: Limit
        in: body
        required: false
        default: 20
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

// bodyOf returns the decoded JSON body of one recorded request.
func bodyOf(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("the request body is not JSON: %v", err)
	}
	return out
}

// fullOffsetPage returns a full page of two items, numbered from n.
func fullOffsetPage(n int) string {
	return fmt.Sprintf(`{"items":[{"id":"doc-%d"},{"id":"doc-%d"}]}`, 2*n+1, 2*n+2)
}

// TestBodyOffsetPagingStopsAtMaxPages replays 25 full body pages against an offset read capped at 20. The engine must
// stop at the cap with the 20 pages it has, a continuation naming offset 40, and one body per page that advances the
// offset by the page size.
func TestBodyOffsetPagingStopsAtMaxPages(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, bodyOffsetYAML))
	replies := make([]replay.Reply, 0, 25)
	for i := 0; i < 25; i++ {
		replies = append(replies, page(fullOffsetPage(i)))
	}
	stub.Script("POST", "/v1/docs/page", replies...)
	rec, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(map[string]any{"limit": 2}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := len(stub.Requests()); got != 20 {
		t.Fatalf("got %d requests; want 20 at the page cap", got)
	}
	if len(rec.Items()) != 40 {
		t.Fatalf("got %d items; want the 40 of the 20 pages", len(rec.Items()))
	}
	p, paged := rec.Page()
	if !paged || !p.HasMore || p.NextOffset != 40 {
		t.Fatalf("the capped read must continue at offset 40: %+v paged=%v", p, paged)
	}
	for i, req := range stub.Requests() {
		if req.Method != "POST" {
			t.Fatalf("request %d method = %q; want POST", i, req.Method)
		}
		if req.Query != "" {
			t.Fatalf("request %d carries query %q; a body-paged read sends no paging query", i, req.Query)
		}
		got := bodyOf(t, req.Body)
		if got["limit"] != float64(2) || got["offset"] != float64(2*i) {
			t.Fatalf("request %d body = %v; want limit 2 and offset %d", i, got, 2*i)
		}
	}
}

// TestBodyPagingStopsWithoutProgress replays a body cursor that echoes the cursor it was sent, so the read never
// advances. The engine must stop after the second page instead of running to the cap.
func TestBodyPagingStopsWithoutProgress(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, bodyCursorYAML))
	stub.Script("POST", "/v1/docs/changes",
		page(`{"items":[{"id":"doc-1"}],"next_cursor":"c2"}`),
		page(`{"items":[{"id":"doc-2"}],"next_cursor":"c2"}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.changes", domain.NewArgs(map[string]any{"limit": 20}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := len(stub.Requests()); got != 2 {
		t.Fatalf("got %d requests; want 2, stopping on the repeated cursor", got)
	}
	if len(rec.Items()) != 2 {
		t.Fatalf("got %d items; want the 2 of the pages read", len(rec.Items()))
	}
}

// TestBodyCursorPaging replays two body cursor pages, the last one ending the read. The engine must collect both
// pages, send the cursor of the second page in its body, and carry no continuation.
func TestBodyCursorPaging(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, bodyCursorYAML))
	stub.Script("POST", "/v1/docs/changes",
		page(`{"items":[{"id":"doc-1"}],"next_cursor":"c2"}`),
		page(`{"items":[{"id":"doc-2"}],"next_cursor":null}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.changes", domain.NewArgs(map[string]any{"limit": 20}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 2 || len(stub.Requests()) != 2 {
		t.Fatalf("got %d items over %d requests; want 2 over 2", len(rec.Items()), len(stub.Requests()))
	}
	if rec.Items()[0].Str("id") != "doc-1" || rec.Items()[1].Str("id") != "doc-2" {
		t.Fatalf("items = %v; want doc-1 and doc-2", rec.Items())
	}
	first := bodyOf(t, stub.Requests()[0].Body)
	if _, has := first["cursor"]; has {
		t.Fatalf("the first body carries a cursor %v; the first page sends none", first)
	}
	second := bodyOf(t, stub.Requests()[1].Body)
	if second["cursor"] != "c2" {
		t.Fatalf("the second body = %v; want cursor c2", second)
	}
	if _, paged := rec.Page(); paged {
		t.Fatal("a cursor read that ends carries no continuation")
	}
}

// TestBodyOffsetAboveTheIntegerCapIsRefused checks that an offset at 2^31 is refused before any request, so the page
// offset stays a non-negative integer below 2^31.
func TestBodyOffsetAboveTheIntegerCapIsRefused(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, bodyOffsetYAML))
	_, err := eng.Read(context.Background(), "example-wiki.docs.page",
		domain.NewArgs(map[string]any{"limit": 2, "offset": 1 << 31}))
	if !errors.Is(err, domain.ErrInvalidOffset) {
		t.Fatalf("Read = %v; want the invalid offset error", err)
	}
	if got := len(stub.Requests()); got != 0 {
		t.Fatalf("got %d requests; want none before the refusal", got)
	}
}
