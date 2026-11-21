package unit_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The POST read of the bindings sends its declared slot as its JSON body. Its query parameters may travel under wire keys
// of their own.

// postQueryOp is a read sent as a POST, with one declared body slot named query.
const postQueryOp = `  - id: example-wiki.docs.query
    summary: Search documents by a JSON query.
    access: read
    method: POST
    read_over_post: true
    path: /docs/query
    params:
      - name: query
        type: Text
        in: body
        required: true
    body:
      query: "{{query}}"
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          query: first
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// postIDsOp is a read sent as a POST whose body slot is a list of IDs. A list that the slot fills adds a level of nesting,
// and the body below nests the slot eight levels deep, so the list is the ninth.
const postIDsOp = `  - id: example-wiki.docs.nested
    summary: Look up documents by a nested list of IDs.
    access: read
    method: POST
    read_over_post: true
    path: /docs/nested
    params:
      - name: ids
        type: IDList
        in: body
        required: true
    body: {a: {b: {c: {d: {e: {f: {g: {h: "{{ids}}"}}}}}}}}
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          ids: [doc-1]
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// postAliasOp is a read whose limit parameter travels under the wire key maxResults.
const postAliasOp = `  - id: example-wiki.docs.alias
    summary: List documents under an aliased page size.
    access: read
    method: GET
    path: /docs
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
        query_key: maxResults
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
        expect:
          id: doc-1
`

// postPagedAliasOp is an offset read whose page size and offset travel under the wire keys pageSize and startAt.
const postPagedAliasOp = `  - id: example-wiki.docs.page
    summary: List one page of documents by offset under aliased keys.
    access: read
    method: GET
    path: /docs/page
    params:
      - name: limit
        type: Limit
        in: query
        required: false
        default: 2
        query_key: pageSize
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

// postCursorAliasOp is a cursor read whose cursor travels under the wire key pageToken.
const postCursorAliasOp = `  - id: example-wiki.docs.list
    summary: List documents by an aliased cursor.
    access: read
    method: GET
    path: /docs
    params:
      - name: cursor
        type: ID
        in: query
        required: false
        query_key: pageToken
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

// postFilterAliasOp is a page-all read whose filter travels under the wire key q, while its paging keys keep their names.
const postFilterAliasOp = `  - id: example-wiki.docs.all
    summary: List every matching document, page by page.
    access: read
    method: GET
    path: /docs
    params:
      - name: filter
        type: Text
        in: query
        required: false
        query_key: q
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
      - request:
          filter: urgent
        response:
          status: 200
          body:
            items:
              - id: doc-1
        expect:
          id: doc-1
`

// postPathOp is a read of one document whose path parameter is a whole segment, used to build a hand-built model that
// gives the path parameter a wire key.
const postPathOp = `  - id: example-wiki.docs.get
    summary: Read one document.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
        expect:
          title: First document
`

// postBuildErr returns the error that the engine gives when it is built from a model, so a hand-built model that breaks a
// rule is shown to be refused. The test fails when the engine is built.
func postBuildErr(t *testing.T, s *spec.Spec) error {
	t.Helper()
	ports := engine.Ports{
		Credentials: &rebindCredentials{},
		Identities:  rebindIdentities{},
		Clock:       &rebindClock{},
		Transport:   rebindReplies(rebindJSON(http.StatusOK, `{}`)),
	}
	_, err := engine.New(s, rebindSource(t), ports)
	return err
}

// TestPostReads_SendsDeclaredSlotOnce binds a POST read to one request whose body holds the slot as JSON, with the fixed
// JSON content type.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: A POST read sends its declared slot once as a JSON value with the fixed JSON content type
func TestPostReads_SendsDeclaredSlotOnce(t *testing.T) {
	s := rebindSpec(t, postQueryOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 1), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.query", map[string]any{"query": "first"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want exactly one", len(reqs))
	}
	var body map[string]any
	if err := json.Unmarshal(reqs[0].body, &body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}
	if body["query"] != "first" {
		t.Fatalf("the body holds %v, want the slot as a JSON value", body["query"])
	}
	if got := reqs[0].header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("the content type is %q, want application/json", got)
	}
}

// TestPostReads_UndeclaredArgumentRefused binds a POST read with an argument the spec does not declare to a refusal before
// any request.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: A POST read with an argument the spec does not declare is refused before any request
func TestPostReads_UndeclaredArgumentRefused(t *testing.T) {
	s := rebindSpec(t, postQueryOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 1), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.query", map[string]any{"query": "first", "extra": "x"})
	if !errors.Is(err, domain.ErrUnknownArgument) {
		t.Fatalf("the error is %v, want the refused argument", err)
	}
	if n := len(tr.seen()); n != 0 {
		t.Fatalf("%d requests were sent, want none", n)
	}
}

// TestPostReads_BodyOneByteOverLimitRefused binds a rendered POST body one byte over 64 KiB to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: A POST read body one byte over 64 KiB is refused before any request
func TestPostReads_BodyOneByteOverLimitRefused(t *testing.T) {
	s := rebindSpec(t, postQueryOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 1), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	// The rendered body is {"query":"<text>"}, which is 12 bytes around the text, so the text takes the rest of the 64 KiB.
	empty, _ := json.Marshal(map[string]any{"query": ""})
	text := strings.Repeat("a", 64*1024+1-len(empty))
	raw, _ := json.Marshal(map[string]any{"query": text})
	if len(raw) != 64*1024+1 {
		t.Fatalf("the rendered body is %d bytes, want 64 KiB plus one", len(raw))
	}
	_, err := pagingRead(eng, "example-wiki.docs.query", map[string]any{"query": text})
	if !errors.Is(err, engine.ErrBodyLimit) {
		t.Fatalf("the error is %v, want the body limit", err)
	}
	if n := len(tr.seen()); n != 0 {
		t.Fatalf("%d requests were sent, want none", n)
	}
}

// TestPostReads_BodyNestingPastEightRefused binds a POST body whose list slot adds a ninth level of nesting to a refusal
// before any request.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: A POST read whose body nests past eight levels is refused before any request
func TestPostReads_BodyNestingPastEightRefused(t *testing.T) {
	s := rebindSpec(t, postIDsOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 1), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.nested", map[string]any{"ids": []string{"doc-1"}})
	if !errors.Is(err, engine.ErrBodyLimit) {
		t.Fatalf("the error is %v, want the body limit", err)
	}
	if n := len(tr.seen()); n != 0 {
		t.Fatalf("%d requests were sent, want none", n)
	}
}

// TestPostReads_HandBuiltModelRefusedAtBuild binds a hand-built model that breaks the access matrix or the POST read rules
// to a refusal when the engine is built.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: A hand-built model that breaks the access matrix or the POST read rules is refused when the engine is built
func TestPostReads_HandBuiltModelRefusedAtBuild(t *testing.T) {
	rows := []struct {
		name   string
		mutate func(s *spec.Spec)
	}{
		{name: "an access label that the matrix refuses", mutate: func(s *spec.Spec) {
			s.Operations[0].Access = "admin"
		}},
		{name: "a POST read that breaks its rules", mutate: func(s *spec.Spec) {
			s.Operations[0].Headers = map[string]string{"Content-Type": "text/plain"}
		}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			s := rebindSpec(t, postQueryOp)
			row.mutate(s)
			if err := postBuildErr(t, s); err == nil {
				t.Fatalf("the engine is built from the model, want a refusal")
			}
		})
	}
}

// TestPostReads_AliasedQueryParameterTravelsUnderItsKey binds a query parameter with a wire key of its own to the key on the
// request, and to an example that still binds through the replay stub.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: A query parameter with a wire key of its own travels under that key and the example still binds
func TestPostReads_AliasedQueryParameterTravelsUnderItsKey(t *testing.T) {
	s := rebindSpec(t, postAliasOp)
	stub := rebindStub(t, s)
	eng := rebindEngine(t, s, stub, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.alias", map[string]any{"limit": 2})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || !strings.Contains(reqs[0].Query, "maxResults=2") || strings.Contains(reqs[0].Query, "limit=") {
		t.Fatalf("the request query is %q, want the wire key maxResults", reqs[0].Query)
	}
	if got := strings.Join(rebindIDs(rec), ","); got != "doc-1" {
		t.Fatalf("the mapped record is %s, want the example's document", got)
	}
}

// TestPostReads_AliasedOffsetStopsAtShortPage binds an offset read with aliased keys to the aliased offset of each request
// and to a stop at the short page.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: An aliased offset read advances the aliased offset and stops at the short page
func TestPostReads_AliasedOffsetStopsAtShortPage(t *testing.T) {
	s := rebindSpec(t, postPagedAliasOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil), pagingPage(pagingDocs(3, 3), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"limit": 2}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want the two pages up to the short page", len(reqs))
	}
	for i, want := range []string{"0", "2"} {
		q := reqs[i].url.Query()
		if q.Get("startAt") != want || q.Get("pageSize") != "2" {
			t.Fatalf("request %d carries startAt %q and pageSize %q, want %s and 2", i, q.Get("startAt"), q.Get("pageSize"), want)
		}
	}
}

// TestPostReads_AliasedCursorNeverSentUnderCatalogName binds an aliased cursor to the second request under its alias only.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: An aliased cursor read sends the cursor under its alias and never under the catalog name
func TestPostReads_AliasedCursorNeverSentUnderCatalogName(t *testing.T) {
	s := rebindSpec(t, postCursorAliasOp)
	tr := rebindReplies(
		pagingPage(pagingDocs(1, 1), map[string]any{"next_cursor": "c2"}),
		pagingPage(pagingDocs(2, 2), map[string]any{"next_cursor": nil}),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.list", nil); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want 2", len(reqs))
	}
	q := reqs[1].url.Query()
	if q.Get("pageToken") != "c2" || q.Has("cursor") {
		t.Fatalf("the second request carries pageToken %q and cursor %v, want the alias only", q.Get("pageToken"), q.Has("cursor"))
	}
}

// TestPostReads_AliasedFilterTravelsUnderAlias binds an aliased filter of a page-all read to its alias, while the fixed
// paging keys keep the names limit and offset.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: An aliased filter in a page-all read travels under its alias while the fixed paging keys keep their names
func TestPostReads_AliasedFilterTravelsUnderAlias(t *testing.T) {
	s := rebindSpec(t, postFilterAliasOp)
	tr := rebindReplies(pagingPage(pagingDocs(1, 2), nil), pagingPage(pagingDocs(3, 3), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.all", map[string]any{"filter": "urgent"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) == 0 {
		t.Fatalf("no request was sent")
	}
	q := reqs[0].url.Query()
	if q.Get("q") != "urgent" || q.Has("filter") {
		t.Fatalf("the filter travels as q %q and as filter %v, want q only", q.Get("q"), q.Has("filter"))
	}
	if !q.Has("limit") || !q.Has("offset") {
		t.Fatalf("the fixed paging keys limit and offset are missing from %q", reqs[0].url.RawQuery)
	}
}

// TestPostReads_HandBuiltPathAliasRefusedAtBuild binds a hand-built model whose path parameter has a wire key to a refusal
// when the engine is built.
// @scenario specs/libs/retriever-engine/behaviours/post-reads-and-query-keys.feature: A hand-built model with an alias on a path parameter is refused when the engine is built
func TestPostReads_HandBuiltPathAliasRefusedAtBuild(t *testing.T) {
	s := rebindSpec(t, postPathOp)
	s.Operations[0].Params[0].QueryKey = "docId"
	if err := postBuildErr(t, s); err == nil {
		t.Fatalf("the engine is built from a path alias, want a refusal")
	}
}
