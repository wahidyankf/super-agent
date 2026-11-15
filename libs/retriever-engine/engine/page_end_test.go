package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// pageEndYAML is an offset read with the page-end rule: the isLast flag decides last, and a page holding more than
// the limit is cut to it. It names example hosts only.
const pageEndYAML = `version: 1
kind: example-wiki
summary: Lists projects with the page-end rule.
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
  - id: example-wiki.projects.list
    summary: List projects.
    access: read
    method: GET
    path: /projects
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
      last_pointer: /isLast
      cut_to_limit: limit
    response:
      items: /values
      fields:
        key:
          pointer: /key
        name:
          pointer: /name
    examples:
      - request:
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            values:
              - key: EXAMPLE-1
                name: First
              - key: EXAMPLE-2
                name: Second
            isLast: false
        expect:
          key: EXAMPLE-1
          name: First
`

// truncatedYAML is a cursor read with the truncated flag: each item tells whether more follows. It names example
// hosts only.
const truncatedYAML = `version: 1
kind: example-wiki
summary: Searches issues with the truncated flag.
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
  - id: example-wiki.issues.search
    summary: Search issues.
    access: read
    method: POST
    read_over_post: true
    path: /search
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
      - name: pageToken
        type: ID
        in: query
        required: false
      - name: jql
        type: Query
        in: body
        required: true
    body:
      jql: "{{jql}}"
    pagination:
      style: cursor
      max_pages: 1
      cursor_param: pageToken
      next_pointer: /nextPageToken
      last_pointer: /isLast
    response:
      items: /issues
      truncated:
        token: /nextPageToken
        last: /isLast
      fields:
        key:
          pointer: /key
    examples:
      - request:
          limit: 2
          jql: project = EXAMPLE
        response:
          status: 200
          body:
            issues:
              - key: EXAMPLE-1
            nextPageToken: token-1
            isLast: false
        expect:
          key: EXAMPLE-1
          truncated: true
`

// TestLastPointerFullPageIsLast checks that a full page with isLast true claims no further page: the flag beats the
// short-page rule, so one request answers both items with no continuation.
func TestLastPointerFullPageIsLast(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pageEndYAML))
	stub.Script("GET", "/v1/projects?limit=2&offset=0",
		page(`{"values":[{"key":"EXAMPLE-1","name":"First"},{"key":"EXAMPLE-2","name":"Second"}],"isLast":true}`))
	rec, err := eng.Read(context.Background(), "example-wiki.projects.list", domain.NewArgs(map[string]any{"limit": 2}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 2 {
		t.Fatalf("got %d items; want 2", len(rec.Items()))
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests; want 1, because a last page claims no further page", n)
	}
	p, paged := rec.Page()
	if !paged || p.HasMore {
		t.Fatalf("a last page must carry no continuation: %+v paged=%v", p, paged)
	}
}

// TestLastPointerMissingIsUnexpected checks that a page with no isLast flag fails with unexpected_response naming the
// pointer path: the flag is required, so a missing one is a shape the rule does not read.
func TestLastPointerMissingIsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pageEndYAML))
	stub.Script("GET", "/v1/projects?limit=2&offset=0",
		page(`{"values":[{"key":"EXAMPLE-1","name":"First"}]}`))
	_, err := eng.Read(context.Background(), "example-wiki.projects.list", domain.NewArgs(map[string]any{"limit": 2}))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if !strings.Contains(err.Error(), "/isLast") {
		t.Fatalf("the refusal does not name the pointer path: %q", err.Error())
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests; want the 1 that answered without the flag", n)
	}
}

// TestCutToLimitKeepsFirstN checks that a page holding more items than the limit keeps the first ones and claims more:
// three items at a limit of two keep the first two and fetch the next page, so no item is lost from the next page.
func TestCutToLimitKeepsFirstN(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pageEndYAML))
	stub.Script("GET", "/v1/projects?limit=2&offset=0",
		page(`{"values":[{"key":"EXAMPLE-1","name":"First"},{"key":"EXAMPLE-2","name":"Second"},{"key":"EXAMPLE-3","name":"Third"}],"isLast":true}`))
	stub.Script("GET", "/v1/projects?limit=2&offset=2",
		page(`{"values":[{"key":"EXAMPLE-4","name":"Fourth"}],"isLast":true}`))
	rec, err := eng.Read(context.Background(), "example-wiki.projects.list", domain.NewArgs(map[string]any{"limit": 2}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	items := rec.Items()
	if len(items) != 3 || items[0].Str("key") != "EXAMPLE-1" || items[1].Str("key") != "EXAMPLE-2" ||
		items[2].Str("key") != "EXAMPLE-4" {
		t.Fatalf("the cut did not keep the first two items and continue: %+v", items)
	}
	if n := len(stub.Requests()); n != 2 {
		t.Fatalf("got %d requests; want 2, because the cut page claims more", n)
	}
}

// TestTruncatedNeedsTokenAndNotLast checks that an item of a page with a token and no last flag is marked truncated:
// the token is present and non-empty and last is false, so every item carries truncated true.
func TestTruncatedNeedsTokenAndNotLast(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, truncatedYAML))
	stub.Script("POST", "/v1/search?limit=2",
		page(`{"issues":[{"key":"EXAMPLE-1"},{"key":"EXAMPLE-2"}],"nextPageToken":"token-1","isLast":false}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.search", domain.NewArgs(map[string]any{"limit": 2, "jql": "project = EXAMPLE"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	items := rec.Items()
	if len(items) != 2 {
		t.Fatalf("got %d items; want 2", len(items))
	}
	for i, item := range items {
		v, ok := item.Get("truncated")
		if !ok || v != true {
			t.Fatalf("item %d truncated = %v, want true", i, v)
		}
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests; want 1", n)
	}
}

// TestTruncatedMissingTokenIsFalse checks that an item of a page with no token is not marked truncated: a missing
// token counts as empty, and a missing last flag counts as false, as the Go code reads them.
func TestTruncatedMissingTokenIsFalse(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, truncatedYAML))
	stub.Script("POST", "/v1/search?limit=2",
		page(`{"issues":[{"key":"EXAMPLE-1"}],"isLast":false}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.search", domain.NewArgs(map[string]any{"limit": 2, "jql": "project = EXAMPLE"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	items := rec.Items()
	if len(items) != 1 {
		t.Fatalf("got %d items; want 1", len(items))
	}
	v, ok := items[0].Get("truncated")
	if !ok || v != false {
		t.Fatalf("truncated = %v, want false without a token", v)
	}
}

// TestTruncatedRefusesAWrongShape checks that a token that is not a string and a last flag that is not a bool fail
// with unexpected_response naming the pointer path: the engine never guesses the shape of the page end.
func TestTruncatedRefusesAWrongShape(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, truncatedYAML))
	args := domain.NewArgs(map[string]any{"limit": 2, "jql": "project = EXAMPLE"})
	stub.Script("POST", "/v1/search?limit=2",
		page(`{"issues":[{"key":"EXAMPLE-1"}],"nextPageToken":7,"isLast":false}`))
	_, err := eng.Read(context.Background(), "example-wiki.issues.search", args)
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "/nextPageToken") {
		t.Fatalf("a non-string token must fail naming its path: %v", err)
	}
	stub.Script("POST", "/v1/search?limit=2",
		page(`{"issues":[{"key":"EXAMPLE-1"}],"nextPageToken":"token-1","isLast":"yes"}`))
	_, err = eng.Read(context.Background(), "example-wiki.issues.search", args)
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "/isLast") {
		t.Fatalf("a non-bool last flag must fail naming its path: %v", err)
	}
}

// TestLastPointerRefusesAWrongShape checks that a last flag that is not a bool fails an offset read with
// unexpected_response naming the pointer path: null counts as missing here, because the flag decides the walk.
func TestLastPointerRefusesAWrongShape(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, pageEndYAML))
	args := domain.NewArgs(map[string]any{"limit": 2})
	stub.Script("GET", "/v1/projects?limit=2&offset=0",
		page(`{"values":[{"key":"EXAMPLE-1","name":"First"}],"isLast":"yes"}`))
	_, err := eng.Read(context.Background(), "example-wiki.projects.list", args)
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "/isLast") {
		t.Fatalf("a non-bool last flag must fail naming its path: %v", err)
	}
	stub.Script("GET", "/v1/projects?limit=2&offset=0",
		page(`{"values":[{"key":"EXAMPLE-1","name":"First"}],"isLast":null}`))
	_, err = eng.Read(context.Background(), "example-wiki.projects.list", args)
	if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "/isLast") {
		t.Fatalf("a null last flag must fail naming its path: %v", err)
	}
}

// TestCutToLimitOnCursorKeepsFirstN checks that the cut also applies to a cursor read: a page holding more items
// than the limit keeps the first ones and marks each item truncated, even with no token and a last flag.
func TestCutToLimitOnCursorKeepsFirstN(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, truncatedCursorCutYAML(t)))
	stub.Script("POST", "/v1/search?limit=2",
		page(`{"issues":[{"key":"EXAMPLE-1"},{"key":"EXAMPLE-2"},{"key":"EXAMPLE-3"}],"isLast":true}`))
	rec, err := eng.Read(context.Background(), "example-wiki.issues.search", domain.NewArgs(map[string]any{"limit": 2, "jql": "project = EXAMPLE"}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	items := rec.Items()
	if len(items) != 2 || items[0].Str("key") != "EXAMPLE-1" || items[1].Str("key") != "EXAMPLE-2" {
		t.Fatalf("the cut did not keep the first two items: %+v", items)
	}
	for i, item := range items {
		v, ok := item.Get("truncated")
		if !ok || v != true {
			t.Fatalf("item %d truncated = %v, want true after the cut", i, v)
		}
	}
}

// truncatedCursorCutYAML returns the truncated fixture with the cut_to_limit rule added, so the cursor read cuts a
// page holding more than the limit.
func truncatedCursorCutYAML(t *testing.T) string {
	t.Helper()
	doc := truncatedYAML
	from := "      last_pointer: /isLast\n"
	i := strings.Index(doc, from)
	if i < 0 {
		t.Fatal("the truncated fixture has no last_pointer line to extend")
	}
	return doc[:i+len(from)] + "      cut_to_limit: limit\n" + doc[i+len(from):]
}
