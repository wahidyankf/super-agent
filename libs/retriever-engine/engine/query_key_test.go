package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// queryKeyOffsetYAML is an offset read whose page parameters travel under wire aliases: limit as maxResults and
// offset as startAt. The stub and the examples keep the catalog names. It names example hosts only.
const queryKeyOffsetYAML = `version: 1
kind: example-wiki
summary: Searches issues with wire aliases for the page parameters.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.issues.search
    summary: Search issues.
    access: read
    method: GET
    path: /issues/search
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
        query_key: maxResults
      - name: offset
        type: Offset
        in: query
        required: false
        default: 0
        query_key: startAt
    pagination:
      style: offset
      max_pages: 5
      limit_param: limit
      offset_param: offset
    response:
      items: /issues
      fields:
        key:
          pointer: /key
    examples:
      - request:
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            issues:
              - key: EXAMPLE-1
        expect:
          key: EXAMPLE-1
`

// queryKeyCursorYAML is a cursor read whose cursor travels under a wire alias: pageToken as nextPage. It names
// example hosts only.
const queryKeyCursorYAML = `version: 1
kind: example-wiki
summary: Lists documents with a wire alias for the cursor.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.list
    summary: List documents.
    access: read
    method: GET
    path: /docs
    params:
      - name: pageToken
        type: ID
        in: query
        required: false
        query_key: nextPage
      - name: limit
        type: Limit
        in: query
        required: false
        default: 20
    pagination:
      style: cursor
      max_pages: 5
      cursor_param: pageToken
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

// queryKeyPageAllYAML is a page-all read with an aliased filter: query travels as jql while the fixed paging keys
// stay limit and offset. It names example hosts only.
const queryKeyPageAllYAML = `version: 1
kind: example-wiki
summary: Searches every document with a wire alias for the filter.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
operations:
  - id: example-wiki.docs.all
    summary: Search every document.
    access: read
    method: GET
    path: /docs/all
    params:
      - name: query
        type: Query
        in: query
        required: true
        query_key: jql
    pagination:
      style: page-all
      max_pages: 3
      page_size: 2
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          query: project = EXAMPLE
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// TestQueryKeyTravelsUnderTheAliasServesTheExample reads the aliased offset operation through the replay stub. The
// example binds by catalog name, the wire carries the aliases, and the mapped record is unchanged.
func TestQueryKeyTravelsUnderTheAliasServesTheExample(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, queryKeyOffsetYAML))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.search",
		domain.NewArgs(map[string]any{"limit": 2, "offset": 0}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 1 || rec.Items()[0].Str("key") != "EXAMPLE-1" {
		t.Fatalf("items = %v, want one item with key EXAMPLE-1", rec.Items())
	}
	if got := stub.Requests()[0].Query; got != "maxResults=2&startAt=0" {
		t.Fatalf("query = %q, want the wire aliases maxResults=2&startAt=0", got)
	}
}

// TestQueryKeyAliasServesOffsetPaging walks two aliased offset pages. The engine must advance the aliased offset and
// stop at the short page, sending maxResults and startAt on every request.
func TestQueryKeyAliasServesOffsetPaging(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, queryKeyOffsetYAML))
	stub.Script("GET", "/v1/issues/search?maxResults=2&startAt=0",
		page(`{"issues":[{"key":"EXAMPLE-1"},{"key":"EXAMPLE-2"}]}`))
	stub.Script("GET", "/v1/issues/search?maxResults=2&startAt=2",
		page(`{"issues":[{"key":"EXAMPLE-3"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.search", domain.NewArgs(map[string]any{"limit": 2}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 3 {
		t.Fatalf("got %d items, want 3", len(rec.Items()))
	}
	for i, req := range stub.Requests() {
		if strings.Contains(req.Query, "limit=") || strings.Contains(req.Query, "offset=") {
			t.Fatalf("request %d carries a catalog name under %q", i, req.Query)
		}
	}
	if got := stub.Requests()[1].Query; got != "maxResults=2&startAt=2" {
		t.Fatalf("second page query = %q, want maxResults=2&startAt=2", got)
	}
}

// TestQueryKeyAliasServesCursorPaging walks two aliased cursor pages. The second request must carry the cursor under
// its alias nextPage, never under the catalog name pageToken.
func TestQueryKeyAliasServesCursorPaging(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, queryKeyCursorYAML))
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1"}],"next_cursor":"c2"}`))
	stub.Script("GET", "/v1/docs?limit=20&nextPage=c2", page(`{"items":[{"id":"doc-2"}],"next_cursor":null}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 2 || len(stub.Requests()) != 2 {
		t.Fatalf("got %d items over %d requests, want 2 over 2", len(rec.Items()), len(stub.Requests()))
	}
	if got := stub.Requests()[1].Query; got != "limit=20&nextPage=c2" {
		t.Fatalf("second page query = %q, want limit=20&nextPage=c2", got)
	}
}

// TestQueryKeyAliasServesPageAllFilters reads every page-all page with an aliased filter. The filter travels as jql
// while the fixed paging keys stay limit and offset.
func TestQueryKeyAliasServesPageAllFilters(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, queryKeyPageAllYAML))
	stub.Script("GET", "/v1/docs/all?jql=project+%3D+EXAMPLE&limit=2&offset=0",
		page(`{"items":[{"id":"doc-1"},{"id":"doc-2"}]}`))
	stub.Script("GET", "/v1/docs/all?jql=project+%3D+EXAMPLE&limit=2&offset=2",
		page(`{"items":[{"id":"doc-3"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.all",
		domain.NewArgs(map[string]any{"query": "project = EXAMPLE"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 3 {
		t.Fatalf("got %d items, want 3", len(rec.Items()))
	}
	for i, req := range stub.Requests() {
		if strings.Contains(req.Query, "query=") {
			t.Fatalf("request %d carries the catalog name under %q", i, req.Query)
		}
	}
	if got := stub.Requests()[0].Query; got != "jql=project+%3D+EXAMPLE&limit=2&offset=0" {
		t.Fatalf("first page query = %q, want the aliased filter with the fixed paging keys", got)
	}
}

// TestNewRefusesAHandBuiltAliasModel checks that the engine constructor refuses a model that the spec would refuse:
// an alias on a path parameter never reaches a request.
func TestNewRefusesAHandBuiltAliasModel(t *testing.T) {
	s := loadSpecFrom(t, queryKeyOffsetYAML)
	s.Operations[0].Params[0].In = "path"
	if _, err := engine.New(s, testSource(t), stubPortsFor(t, loadSpecFrom(t, queryKeyOffsetYAML))); !errors.Is(err, spec.ErrInvalid) {
		t.Fatalf("err = %v, want the spec's invalid error", err)
	}
}
