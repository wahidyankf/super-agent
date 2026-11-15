package engine_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// pageAllYAML is a read over POST that pages with no declared page parameters: the rule names the fixed page size,
// the page cap, and the body as the location of the paging keys. It names example hosts only.
const pageAllYAML = `version: 1
kind: example-wiki
summary: Lists every document of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.all
    summary: List every document.
    access: read
    method: POST
    read_over_post: true
    path: /docs/all
    body: {}
    pagination:
      style: page-all
      in: body
      max_pages: 3
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

// pageAllQueryYAML pages the same way in the query: the paging keys travel as query parameters.
var pageAllQueryYAML = strings.Replace(strings.Replace(pageAllYAML, "example-wiki.docs.all", "example-wiki.docs.query", 1),
	"/docs/all", "/docs/query", 1)

// fullPageAllPage returns a full page-all page of two items, numbered from n.
func fullPageAllPage(n int) string {
	return fmt.Sprintf(`{"items":[{"id":"doc-%d"},{"id":"doc-%d"}]}`, 2*n+1, 2*n+2)
}

// TestPageAllReadsEveryPageUntilAShortPage replays two full body pages and a short one against a page-all read. The
// engine must collect all five items, send the fixed paging keys in each JSON body, and carry no continuation.
func TestPageAllReadsEveryPageUntilAShortPage(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pageAllYAML))
	stub.Script("POST", "/v1/docs/all",
		page(fullPageAllPage(0)),
		page(fullPageAllPage(1)),
		page(`{"items":[{"id":"doc-5"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.all", domain.NewArgs(map[string]any{}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := len(stub.Requests()); got != 3 {
		t.Fatalf("got %d requests; want 3, stopping at the short page", got)
	}
	if len(rec.Items()) != 5 {
		t.Fatalf("got %d items; want the 5 of the 3 pages", len(rec.Items()))
	}
	for i, req := range stub.Requests() {
		if req.Query != "" {
			t.Fatalf("request %d carries query %q; a body page-all read sends no paging query", i, req.Query)
		}
		got := bodyOf(t, req.Body)
		if got["limit"] != float64(2) || got["offset"] != float64(2*i) {
			t.Fatalf("request %d body = %v; want limit 2 and offset %d", i, got, 2*i)
		}
	}
}

// TestPageAllFailsAtTheCapWithAPageCapError replays full pages past the cap of 3 against a page-all read. The
// engine must fail with the page-cap error after the third page, mirroring the Go reader, instead of returning the
// pages it has.
func TestPageAllFailsAtTheCapWithAPageCapError(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pageAllYAML))
	stub.Script("POST", "/v1/docs/all",
		page(fullPageAllPage(0)),
		page(fullPageAllPage(1)),
		page(fullPageAllPage(2)),
		page(fullPageAllPage(3)))
	_, err := eng.Read(context.Background(), "example-wiki.docs.all", domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrPageCapReached) {
		t.Fatalf("err = %v, want ErrPageCapReached", err)
	}
	if got := len(stub.Requests()); got != 3 {
		t.Fatalf("got %d requests; want 3, failing before a fourth page", got)
	}
}

// TestPageAllSendsItsPagingKeysInTheQuery replays two query pages against a page-all read with no body location. The
// engine must carry the fixed offset and limit as query parameters and collect both pages.
func TestPageAllSendsItsPagingKeysInTheQuery(t *testing.T) {
	doc := strings.Replace(pageAllQueryYAML, "      in: body\n", "", 1)
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	stub.Script("POST", "/v1/docs/query?limit=2&offset=0", page(fullPageAllPage(0)))
	stub.Script("POST", "/v1/docs/query?limit=2&offset=2", page(`{"items":[{"id":"doc-3"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.query", domain.NewArgs(map[string]any{}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 3 {
		t.Fatalf("got %d items; want the 3 of the 2 pages", len(rec.Items()))
	}
	for i, req := range stub.Requests() {
		want := fmt.Sprintf("limit=2&offset=%d", 2*i)
		if req.Query != want {
			t.Fatalf("request %d query = %q; want %q", i, req.Query, want)
		}
	}
}

// pageAllDeclaredYAML pages in the body with a declared filter beside the paging keys. A text parameter may even be
// named like a paging key: the paging keys beat a declared key of the same name, so the declared document cannot move
// the walk off its pages. It names example hosts only.
const pageAllDeclaredYAML = `version: 1
kind: example-wiki
summary: Lists filtered documents of an example wiki.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.filtered
    summary: List filtered documents.
    access: read
    method: POST
    read_over_post: true
    path: /docs/filtered
    params:
      - name: filter
        type: Text
        in: body
        required: false
      - name: limit
        type: Text
        in: body
        required: false
    body:
      filter: "{{filter}}"
      limit: "{{limit}}"
    pagination:
      style: page-all
      in: body
      max_pages: 3
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

// TestPageAllKeepsDeclaredBodyKeysAndBeatsPagingNames checks that a page-all body keeps the declared keys and sets
// the paging keys itself: the declared filter travels, and the declared limit gives way to the fixed page size.
func TestPageAllKeepsDeclaredBodyKeysAndBeatsPagingNames(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pageAllDeclaredYAML))
	stub.Script("POST", "/v1/docs/filtered",
		page(fullPageAllPage(0)),
		page(`{"items":[{"id":"doc-5"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.filtered",
		domain.NewArgs(map[string]any{"filter": "handbook", "limit": "once"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 3 {
		t.Fatalf("got %d items; want the 3 of the 2 pages", len(rec.Items()))
	}
	for i, req := range stub.Requests() {
		got := bodyOf(t, req.Body)
		if got["filter"] != "handbook" {
			t.Fatalf("request %d body = %v; want the declared filter", i, got)
		}
		if got["limit"] != float64(2) || got["offset"] != float64(2*i) {
			t.Fatalf("request %d body = %v; want limit 2 and offset %d", i, got, 2*i)
		}
	}
}

// TestPageAllStopsAFailedPage checks that a page-all read fails with the error of a failed page, after the pages it
// already read, instead of paging past it.
func TestPageAllStopsAFailedPage(t *testing.T) {
	doc := strings.Replace(pageAllQueryYAML, "      in: body\n", "", 1)
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	stub.Script("POST", "/v1/docs/query?limit=2&offset=0", page(fullPageAllPage(0)))
	stub.Script("POST", "/v1/docs/query?limit=2&offset=2", replay.Reply{Status: http.StatusInternalServerError})
	_, err := eng.Read(context.Background(), "example-wiki.docs.query", domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrUpstreamFailure) {
		t.Fatalf("err = %v, want ErrUpstreamFailure", err)
	}
	if got := len(stub.Requests()); got != 2 {
		t.Fatalf("got %d requests; want 2, stopping at the failed page", got)
	}
}

// TestPageAllSpacesItsPagesByTheMinDelay checks the request pacing of a page-all read: the engine waits the pacing
// delay before every page after the first, mirroring the Go gateway. The fake clock records the waits.
func TestPageAllSpacesItsPagesByTheMinDelay(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpecFrom(t, pageAllYAML))
	stub.Script("POST", "/v1/docs/all",
		page(fullPageAllPage(0)),
		page(fullPageAllPage(1)),
		page(`{"items":[{"id":"doc-5"}]}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.all", domain.NewArgs(map[string]any{})); err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}
	if got := clock.waits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}

// TestPageAllTopsAShortRetryWaitUpToTheMinDelay checks the pacing of a rate-limited page-all read: a 429 wait shorter
// than the pacing delay is topped up to it, so the retried request keeps the spacing of the Go gateway.
func TestPageAllTopsAShortRetryWaitUpToTheMinDelay(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpecFrom(t, pageAllYAML))
	stub.Script("POST", "/v1/docs/all",
		replay.Reply{Status: http.StatusTooManyRequests, Header: map[string][]string{"Retry-After": {"0"}}},
		page(`{"items":[{"id":"doc-1"}]}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.all", domain.NewArgs(map[string]any{})); err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []time.Duration{100 * time.Millisecond}
	if got := clock.waits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}

// TestPageAllKeepsALongerRetryWait checks that a 429 wait longer than the pacing delay is kept as the reply asks: a
// longer Retry-After already covers the delay, so nothing is added.
func TestPageAllKeepsALongerRetryWait(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpecFrom(t, pageAllYAML))
	stub.Script("POST", "/v1/docs/all",
		replay.Reply{Status: http.StatusTooManyRequests, Header: map[string][]string{"Retry-After": {"1"}}},
		page(`{"items":[{"id":"doc-1"}]}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.all", domain.NewArgs(map[string]any{})); err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []time.Duration{time.Second}
	if got := clock.waits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}

// TestPageAllRefusesAnUnmappablePage checks that a page-all read fails when a page does not hold a list, instead of
// mapping a part of it.
func TestPageAllRefusesAnUnmappablePage(t *testing.T) {
	doc := strings.Replace(pageAllQueryYAML, "      in: body\n", "", 1)
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	stub.Script("POST", "/v1/docs/query?limit=2&offset=0", page(fullPageAllPage(0)))
	stub.Script("POST", "/v1/docs/query?limit=2&offset=2", page(`{"items":{}}`))
	_, err := eng.Read(context.Background(), "example-wiki.docs.query", domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if got := len(stub.Requests()); got != 2 {
		t.Fatalf("got %d requests; want 2, stopping at the unmappable page", got)
	}
}

// TestPageAllRefusesAnOutOfRangePageSizeBeforeAnyRequest checks the engine's closed page-size range on a hand-built
// model: a page-all rule with no page size fails before any request, so a model that skipped validation cannot page
// without bound.
func TestPageAllRefusesAnOutOfRangePageSizeBeforeAnyRequest(t *testing.T) {
	model := loadSpecFrom(t, pageAllYAML)
	for i := range model.Operations {
		model.Operations[i].Pagination.PageSize = 0
	}
	stub, err := replay.New(loadSpecFrom(t, pageAllYAML))
	if err != nil {
		t.Fatalf("build the replay stub: %v", err)
	}
	eng, err := engine.New(model, testSource(t), stubPorts(stub, &fakeClock{}))
	if err != nil {
		t.Fatalf("engine.New: %v", err)
	}
	_, err = eng.Read(context.Background(), "example-wiki.docs.all", domain.NewArgs(map[string]any{}))
	if !errors.Is(err, domain.ErrInvalidLimit) {
		t.Fatalf("err = %v, want ErrInvalidLimit", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused call sent %d requests", n)
	}
}
