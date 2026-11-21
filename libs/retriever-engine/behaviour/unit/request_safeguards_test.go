package unit_test

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// The request safeguards of these bindings build each request from a validated spec. Every request is recorded by the
// in-memory transport, so each test checks the bytes that the engine would have sent.

// requestsHeaderDoc is a single-document read with a constant header and a header slot. The slot takes the note parameter.
const requestsHeaderDoc = `  - id: example-wiki.docs.one
    summary: Read one document with a constant header and a header slot.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
      - name: note
        type: Text
        in: query
        required: false
    headers:
      X-Example-Client: retriever
      X-Note: "{note}"
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

// requestsQueryDoc is a read with one declared query text parameter, q, on the list of documents.
const requestsQueryDoc = `  - id: example-wiki.docs.search
    summary: Search documents by a query text.
    access: read
    method: GET
    path: /docs
    params:
      - name: q
        type: Text
        in: query
        required: false
    response:
      items: /items
      fields:
        id:
          pointer: /id
    examples:
      - request:
          q: urgent
        response:
          status: 200
          body:
            items: [{id: doc-1}]
        expect:
          id: doc-1
`

// requestsOffsetDoc is an offset read with a declared query text parameter, q, beside its paging keys.
const requestsOffsetDoc = `  - id: example-wiki.docs.page
    summary: List one page of documents by offset with a query text.
    access: read
    method: GET
    path: /docs
    params:
      - name: q
        type: Text
        in: query
        required: false
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
          q: urgent
          limit: 2
          offset: 0
        response:
          status: 200
          body:
            items: [{id: doc-1}]
        expect:
          id: doc-1
`

// requestsPostReadDoc is a POST read whose body holds a slot inside a longer text.
const requestsPostReadDoc = `  - id: example-wiki.docs.note
    summary: Read one note sent as a POST body.
    access: read
    method: POST
    read_over_post: true
    path: /notes
    params:
      - name: title
        type: Text
        in: body
        required: true
    body:
      text: "{{title}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: First
        response:
          status: 200
          body:
            id: note-1
        expect:
          id: note-1
`

// requestsPathOp is the path read whose one ID segment takes the caller's value.
const requestsPathOp = `  - id: example-wiki.docs.one
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

// requestsHeaderNames returns the names of a request header, in sorted order.
func requestsHeaderNames(h http.Header) []string {
	var names []string
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestRequestSafeguards_OnlyDeclaredHeadersSent binds a read's request to the declared headers, the accept header, and the
// authorization header.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A request carries only the headers the spec declares, the accept header, and the authorization header
func TestRequestSafeguards_OnlyDeclaredHeadersSent(t *testing.T) {
	s := rebindSpec(t, readsListDoc)
	tr := rebindReplies(pagingPage(pagingDocs(1, 1), map[string]any{"next_cursor": nil}))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.list", map[string]any{"limit": 20}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	got := strings.Join(requestsHeaderNames(reqs[0].header), ",")
	want := "Accept,Authorization,X-Example-Client"
	if got != want {
		t.Fatalf("the request holds headers %s, want %s", got, want)
	}
}

// TestRequestSafeguards_HeaderValueWithLineBreakRefused binds a header slot that takes a value with a line break to a refusal
// before any request.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A header value with a control character is refused before any request
func TestRequestSafeguards_HeaderValueWithLineBreakRefused(t *testing.T) {
	s := rebindSpec(t, requestsHeaderDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1", "note": "line one\nline two"})
	if !errors.Is(err, engine.ErrInvalidHeader) || len(tr.seen()) != 0 {
		t.Fatalf("the error is %v with %d requests, want an invalid header and no request", err, len(tr.seen()))
	}
}

// TestRequestSafeguards_ConstantHeaderSentAndUnsetSlotLeftOut binds a constant header to its value in the request, and an
// unset header slot to no header at all.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A constant header is sent as it is, a slot header takes its value, and an unset slot is left out
func TestRequestSafeguards_ConstantHeaderSentAndUnsetSlotLeftOut(t *testing.T) {
	s := rebindSpec(t, requestsHeaderDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	if got := reqs[0].header.Get("X-Example-Client"); got != "retriever" {
		t.Fatalf("the constant header is %q, want retriever", got)
	}
	if _, present := reqs[0].header["X-Note"]; present {
		t.Fatalf("the unset header slot is sent, want it left out")
	}
}

// TestRequestSafeguards_PathValueLeavingSegmentRefused binds a path value that would leave its segment to a refusal before
// any request.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A path value that would leave its segment is refused before any request
func TestRequestSafeguards_PathValueLeavingSegmentRefused(t *testing.T) {
	rows := []struct{ name, value string }{
		{name: "dot dot", value: "../admin"},
		{name: "line break", value: "a value with a line break\nx"},
		{name: "slash", value: "a value with a slash/x"},
		{name: "percent sign", value: "a value with a percent sign %2e"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			s := rebindSpec(t, requestsPathOp)
			tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": row.value}); err == nil || len(tr.seen()) != 0 {
				t.Fatalf("the call is served with %d requests, want a refusal and no request", len(tr.seen()))
			}
		})
	}
}

// TestRequestSafeguards_PathSlotFillsOneSegment binds a plain path value to the request path as one whole segment.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A path slot fills one whole path segment
func TestRequestSafeguards_PathSlotFillsOneSegment(t *testing.T) {
	s := rebindSpec(t, requestsPathOp)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 || reqs[0].url.Path != "/v1/docs/doc-1" {
		t.Fatalf("the request path is %v, want the value as one whole segment", reqs[0].url)
	}
}

// TestRequestSafeguards_QueryValueEncodedNotSpliced binds a query value with reserved characters to encoded query text, with
// the path unchanged.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A query value with reserved characters travels as encoded query text
func TestRequestSafeguards_QueryValueEncodedNotSpliced(t *testing.T) {
	s := rebindSpec(t, requestsQueryDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"items":[]}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	value := "a&b=c d/e?f#g"
	if _, err := pagingRead(eng, "example-wiki.docs.search", map[string]any{"q": value}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	if got := reqs[0].url.Query().Get("q"); got != value {
		t.Fatalf("the query carries %q, want the value as given", got)
	}
	if reqs[0].url.Path != "/v1/docs" {
		t.Fatalf("the path is %q, want the path with no part of the value", reqs[0].url.Path)
	}
}

// TestRequestSafeguards_DeclaredQueryEncodedPagingValueReplaces binds a declared query value to encoded query text, and the
// paging offset to the value that the engine computes in place of the caller's.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A declared query value is encoded and a paging value replaces the declared one
func TestRequestSafeguards_DeclaredQueryEncodedPagingValueReplaces(t *testing.T) {
	s := rebindSpec(t, requestsOffsetDoc)
	tr := rebindReplies(pagingPage(pagingDocs(1, 1), nil))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.page", map[string]any{"q": "a b&c", "limit": 2, "offset": 99}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	q := reqs[0].url.Query()
	if q.Get("q") != "a b&c" {
		t.Fatalf("the declared query value is %q, want it encoded and intact", q.Get("q"))
	}
	// The caller starts the offset at 99, so the first page carries 99 as its one offset key, in place of the declared value.
	if got := q["offset"]; len(got) != 1 || got[0] != "99" {
		t.Fatalf("the offset is %v, want the one paging value 99", got)
	}
}

// TestRequestSafeguards_BodyValueJSONEncodedAsGiven binds a write body value with quotes, a line break, and text that looks
// like a slot to the text the caller gave.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A body value is JSON-encoded and a value that looks like a slot stays literal text
func TestRequestSafeguards_BodyValueJSONEncodedAsGiven(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	value := "He said \"hi\"\nand {{title}} again"
	if _, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": value})); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	body := writesBody(t, tr)
	if body["title"] != value {
		t.Fatalf("the body carries %q, want the text the caller gave", body["title"])
	}
}

// TestRequestSafeguards_SlotInsideLongerTextIsText binds a slot inside a longer text of the body to plain text. The validator
// refuses such a body at load, so the body is set on the validated model after it loads, which is the only way the renderer
// meets it; the render must keep the text as it is.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A slot inside a longer text in the body is only text
func TestRequestSafeguards_SlotInsideLongerTextIsText(t *testing.T) {
	s := rebindSpec(t, requestsPostReadDoc)
	s.Operations[0].Body = map[string]any{"text": "before {{title}} after"}
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"id":"note-1"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.note", map[string]any{"title": "First"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	body := writesBody(t, tr)
	if body["text"] != "before {{title}} after" {
		t.Fatalf("the body holds %v, want the slot as plain text", body["text"])
	}
}

// TestRequestSafeguards_RedirectRefusedAfterOneRequest binds a reply that redirects to another host to the redirect error,
// with exactly one request.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A redirect in the reply is refused and only one request is sent
func TestRequestSafeguards_RedirectRefusedAfterOneRequest(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	redirect := rebindReply{status: http.StatusFound, header: http.Header{"Location": {"https://elsewhere.example.test/docs/doc-1"}}}
	tr := rebindReplies(redirect)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrRedirected) || len(tr.seen()) != 1 {
		t.Fatalf("the error is %v with %d requests, want the redirect error and exactly one request", err, len(tr.seen()))
	}
}

// TestRequestSafeguards_KeyOnlyInAuthorizationOfRead binds a read with a key to the key in the authorization header and in no
// other part of the request.
// @scenario specs/libs/retriever-engine/behaviours/request-safeguards.feature: A key appears only in the authorization header of a read
func TestRequestSafeguards_KeyOnlyInAuthorizationOfRead(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	eng, stub := readsStubEngine(t, s)
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	if !strings.Contains(reqs[0].Header.Get("Authorization"), rebindKey) {
		t.Fatalf("the key is not in the authorization header")
	}
	for name, values := range reqs[0].Header {
		if name == "Authorization" {
			continue
		}
		if strings.Contains(strings.Join(values, ","), rebindKey) {
			t.Fatalf("the key appears in header %s", name)
		}
	}
	if strings.Contains(reqs[0].Path, rebindKey) || strings.Contains(reqs[0].Query, rebindKey) || strings.Contains(string(reqs[0].Body), rebindKey) {
		t.Fatalf("the key appears in the path, query, or body")
	}
}
