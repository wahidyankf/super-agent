package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The writes of these bindings are sent once, and their outcome is known only when the source said so. A write that failed
// after its request left reports an unknown outcome, and a write that failed before it left reports a known failure.

// writesDocOp is the create write of the example wiki: a POST of one document with a required title in its body. Its reply
// holds a document id and a bool map, so a reply value of another shape fails the mapping.
const writesDocOp = `  - id: example-wiki.docs.create
    summary: Create one document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
        in: body
        required: true
    body:
      title: "{{title}}"
    response:
      fields:
        id:
          pointer: /id
        flags:
          pointer: /flags
          bool_map: true
    examples:
      - request:
          title: First
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`

// writesHeaderDoc is the create write with a constant header and a header slot that takes the note parameter.
const writesHeaderDoc = `  - id: example-wiki.docs.create
    summary: Create one document with a header slot.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
        in: body
        required: true
      - name: note
        type: Text
        in: query
        required: false
    headers:
      X-Example-Client: retriever
      X-Note: "{note}"
    body:
      title: "{{title}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: First
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`

// writesPathDoc is a write of one document by its ID, whose ID is one path segment.
const writesPathDoc = `  - id: example-wiki.docs.update
    summary: Update one document.
    access: write
    method: PUT
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
      - name: title
        type: Text
        in: body
        required: true
    body:
      title: "{{title}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          id: doc-1
          title: First
        response:
          status: 200
          body:
            id: doc-1
        expect:
          id: doc-1
`

// writesOptionalDoc is a create write with an optional body parameter, note, that the body takes only when the caller sets it.
const writesOptionalDoc = `  - id: example-wiki.docs.create
    summary: Create one document with an optional note.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
        in: body
        required: true
      - name: note
        type: Text
        in: body
        required: false
    body:
      title: "{{title}}"
      note: "{{note}}"
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: First
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`

// writesErrorsDoc is a create write whose 409 reply the spec maps to the document conflict reason.
const writesErrorsDoc = `  - id: example-wiki.docs.create
    summary: Create one document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
        in: body
        required: true
    body:
      title: "{{title}}"
    response:
      errors:
        "409": document_conflict
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: First
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`

// writesNoneDoc is a create write whose paging style is none, so it is an unpaged write.
const writesNoneDoc = `  - id: example-wiki.docs.create
    summary: Create one document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
        in: body
        required: true
    body:
      title: "{{title}}"
    pagination:
      style: none
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request:
          title: First
        response:
          status: 201
          body:
            id: doc-9
        expect:
          id: doc-9
`

// writesAdminDoc is an admin operation of the example wiki, which the write path serves and the read path refuses.
const writesAdminDoc = `  - id: example-wiki.admin.reset
    summary: Reset the example wiki.
    access: admin
    method: POST
    path: /admin/reset
    response:
      fields:
        id:
          pointer: /id
    examples:
      - request: {}
        response:
          status: 200
          body:
            id: reset-1
        expect:
          id: reset-1
`

// writesConvertDoc is a create write whose reply field names a converter, the one the engine serves or the one it refuses.
func writesConvertDoc(convert string) string {
	return `  - id: example-wiki.docs.create
    summary: Create one document.
    access: write
    method: POST
    path: /docs
    params:
      - name: title
        type: Text
        in: body
        required: true
    body:
      title: "{{title}}"
    response:
      fields:
        summary:
          pointer: /summary
          convert: ` + convert + `
    examples:
      - request:
          title: First
        response:
          status: 201
          body:
            summary: Hi
        expect:
          summary: Hi
`
}

// writesBody returns the JSON body of the last request that the in-memory transport received.
func writesBody(t *testing.T, tr *rebindTransport) map[string]any {
	t.Helper()
	reqs := tr.seen()
	if len(reqs) == 0 {
		t.Fatalf("no request was sent")
	}
	var body map[string]any
	if err := json.Unmarshal(reqs[len(reqs)-1].body, &body); err != nil {
		t.Fatalf("the body is not JSON: %v", err)
	}
	return body
}

// writesCreate runs one create of the example wiki with the given title, and returns the record and the error.
func writesCreate(t *testing.T, eng *engine.Engine, title string) (domain.Record, error) {
	t.Helper()
	return eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": title}))
}

// writesGuardedEngine builds an engine that reaches the source through the guarded transport over a fake network, so no
// request reaches a socket.
func writesGuardedEngine(t *testing.T, s *spec.Spec, n engine.Network) *engine.Engine {
	t.Helper()
	ports := engine.Ports{
		Credentials: &rebindCredentials{},
		Identities:  rebindIdentities{},
		Clock:       &rebindClock{},
		Network:     n,
	}
	eng, err := engine.New(s, rebindSource(t), ports)
	if err != nil {
		t.Fatalf("build the engine: %v", err)
	}
	return eng
}

// TestWrites_AcceptedWriteSentOnceAndMapsFields binds an accepted write to one request, no second confirmation, and the
// declared fields of the result.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write the source accepts is sent once with no confirmation and maps its declared fields
func TestWrites_AcceptedWriteSentOnceAndMapsFields(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := writesCreate(t, eng, "First")
	if err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	if n := len(tr.seen()); n != 1 {
		t.Fatalf("%d requests, want exactly one with no confirmation", n)
	}
	if rec.Str("id") != "doc-9" {
		t.Fatalf("the result holds id %q, want the declared field", rec.Str("id"))
	}
}

// TestWrites_TimedOutWriteUnknownNotSentAgain binds a write that times out after its request left to an unknown outcome and
// exactly one request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write that times out after its request left reports an unknown outcome and is not sent again
func TestWrites_TimedOutWriteUnknownNotSentAgain(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	tr := rebindReplies(rebindReply{stall: true})
	eng := rebindEngine(t, s, tr, &rebindClock{expired: true})
	_, err := writesCreate(t, eng, "First")
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("the error is %v, want an unknown outcome", err)
	}
	if n := len(tr.seen()); n != 1 {
		t.Fatalf("%d requests, want exactly one", n)
	}
}

// TestWrites_FailureAfterRequestLeftUnknownSentOnce binds each failure that happens after the request left to an unknown
// outcome, with exactly one request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A failure after the request left is an unknown outcome and the write is sent once
func TestWrites_FailureAfterRequestLeftUnknownSentOnce(t *testing.T) {
	rows := []struct {
		failure string
		reply   func() rebindReply
	}{
		{failure: "a reply that is over the reply cap", reply: func() rebindReply {
			return rebindJSON(http.StatusCreated, strings.Repeat(" ", 10<<20+1))
		}},
		{failure: "a reply nested past the depth cap", reply: func() rebindReply {
			return rebindJSON(http.StatusCreated, `{"id":`+strings.Repeat("[", 70)+strings.Repeat("]", 70)+`}`)
		}},
		{failure: "a reply body that breaks while it is read", reply: func() rebindReply {
			rep := rebindJSON(http.StatusCreated, `{"id":"doc-9"}`)
			rep.bodyErr = errors.New("the body broke while it was read")
			return rep
		}},
		{failure: "a connection cut after the request left", reply: func() rebindReply {
			return rebindReply{err: io.ErrUnexpectedEOF}
		}},
		{failure: "an injected transport that fails after sending", reply: func() rebindReply {
			return rebindReply{err: errors.New("the injected transport failed after it sent")}
		}},
		{failure: "a bool map with a value that is not a flag", reply: func() rebindReply {
			return rebindJSON(http.StatusCreated, `{"id":"doc-9","flags":{"alpha":5}}`)
		}},
	}
	for _, row := range rows {
		t.Run(row.failure, func(t *testing.T) {
			s := rebindSpec(t, writesDocOp)
			tr := rebindReplies(row.reply())
			eng := rebindEngine(t, s, tr, &rebindClock{})
			_, err := writesCreate(t, eng, "First")
			if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("the error is %v, want an unknown outcome", err)
			}
			if n := len(tr.seen()); n != 1 {
				t.Fatalf("%d requests, want exactly one", n)
			}
		})
	}
}

// TestWrites_FailureBeforeRequestLeftIsKnown binds each failure that happens before the request leaves to a known failure
// that is not an unknown outcome. A private address is refused before any dial, and the other two rows dial and fail before
// the request is sent.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A failure before the request leaves is a known failure and sends nothing
func TestWrites_FailureBeforeRequestLeftIsKnown(t *testing.T) {
	rows := []struct {
		failure string
		addr    netip.Addr
		dial    func() (net.Conn, error)
		dialed  bool
	}{
		{failure: "a connection that never opens", addr: netip.MustParseAddr("203.0.113.10"),
			dial: func() (net.Conn, error) { return nil, errors.New("connection refused") }, dialed: true},
		{failure: "a TLS handshake that fails", addr: netip.MustParseAddr("203.0.113.10"),
			dial: func() (net.Conn, error) {
				client, server := net.Pipe()
				_ = server.Close()
				return client, nil
			}, dialed: true},
		{failure: "a source name on a private address", addr: netip.AddrFrom4([4]byte{10, 0, 0, 5}),
			dial: func() (net.Conn, error) { return nil, errors.New("dial must not be reached") }, dialed: false},
	}
	for _, row := range rows {
		t.Run(row.failure, func(t *testing.T) {
			s := rebindSpec(t, writesDocOp)
			dials := 0
			n := engine.Network{
				Lookup: func(context.Context, string) ([]netip.Addr, error) { return []netip.Addr{row.addr}, nil },
				Dial: func(context.Context, string, string) (net.Conn, error) {
					dials++
					return row.dial()
				},
			}
			eng := writesGuardedEngine(t, s, n)
			_, err := writesCreate(t, eng, "First")
			if err == nil || errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("the error is %v, want a known failure that is not an unknown outcome", err)
			}
			if (dials > 0) != row.dialed {
				t.Fatalf("the source was dialed %d times, want dialed=%v", dials, row.dialed)
			}
		})
	}
}

// TestWrites_FourXXMappedErrorKeptNotUnknown binds a 4xx reply that the spec maps to a reason to that reason, and not to an
// unknown outcome.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A 4xx reply to a write keeps the error that the spec maps to its status and is never an unknown outcome
func TestWrites_FourXXMappedErrorKeptNotUnknown(t *testing.T) {
	s := rebindSpec(t, writesErrorsDoc)
	tr := rebindReplies(rebindJSON(http.StatusConflict, `{"message":"taken"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := writesCreate(t, eng, "First")
	if !errors.Is(err, domain.ErrDocumentConflict) || errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("the error is %v, want the document conflict and not an unknown outcome", err)
	}
}

// TestWrites_FourXXReplyBodyNotInError binds a 4xx reply that holds a body to an error that does not show the body.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A 4xx reply to a write does not echo its body in the error
func TestWrites_FourXXReplyBodyNotInError(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	tr := rebindReplies(rebindJSON(http.StatusBadRequest, `{"message":"the secret body text"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := writesCreate(t, eng, "First")
	if err == nil || strings.Contains(err.Error(), "the secret body text") {
		t.Fatalf("the error is %v, want a failure that does not show the body", err)
	}
}

// TestWrites_RedirectRefusedAfterOneRequest binds a write answered with a redirect to a refusal after one request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write whose source answers with a redirect is refused after one request
func TestWrites_RedirectRefusedAfterOneRequest(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	redirect := rebindReply{status: http.StatusFound, header: http.Header{"Location": {"https://elsewhere.example.test/docs"}}}
	tr := rebindReplies(redirect)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := writesCreate(t, eng, "First")
	if !errors.Is(err, domain.ErrRedirected) || len(tr.seen()) != 1 {
		t.Fatalf("the error is %v with %d requests, want the redirect refusal after one request", err, len(tr.seen()))
	}
}

// TestWrites_RetriedAfterUsable429ThenAccepted binds a write answered with 429 and a usable wait to one wait and a second
// request that succeeds.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write is retried after a 429 reply as a read is
func TestWrites_RetriedAfterUsable429ThenAccepted(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	tr := rebindReplies(timingRetryAfter("1"), rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	rec, err := writesCreate(t, eng, "First")
	if err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[1s]" || len(tr.seen()) != 2 || rec.Str("id") != "doc-9" {
		t.Fatalf("the waits are %s with %d requests, want one wait, a second send, and the success", got, len(tr.seen()))
	}
}

// TestWrites_KeepsRateLimitedAfterThreeRetries binds a write that keeps meeting 429 to a stop after three retries with exit
// status 124.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write that keeps meeting 429 stops after three retries with the rate-limit exit status
func TestWrites_KeepsRateLimitedAfterThreeRetries(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	tr := rebindReplies(timingRetryAfter("1"))
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	_, err := writesCreate(t, eng, "First")
	if !errors.Is(err, domain.ErrSourceRateLimited) || engine.ExitStatus(err) != engine.ExitTimedOut {
		t.Fatalf("the error is %v with exit status %d, want the rate limit and exit status 124", err, engine.ExitStatus(err))
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[1s 1s 1s]" || len(tr.seen()) != 4 {
		t.Fatalf("the waits are %s with %d requests, want three waits and four requests", got, len(tr.seen()))
	}
}

// TestWrites_HeaderValueWithLineBreakRefused binds a write whose header slot takes a value with a line break to a refusal
// before any request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write header value with a line break is refused before any request
func TestWrites_HeaderValueWithLineBreakRefused(t *testing.T) {
	s := rebindSpec(t, writesHeaderDoc)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := eng.Write(context.Background(), "example-wiki.docs.create",
		domain.NewArgs(map[string]any{"title": "First", "note": "line one\nline two"}))
	if !errors.Is(err, engine.ErrInvalidHeader) || len(tr.seen()) != 0 {
		t.Fatalf("the error is %v with %d requests, want an invalid header and no request", err, len(tr.seen()))
	}
}

// TestWrites_OnlyDeclaredHeadersSent binds a write's request to its declared headers, the accept header, the content type,
// and the authorization header.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write carries only its declared headers, the accept header, the content type, and the authorization header
func TestWrites_OnlyDeclaredHeadersSent(t *testing.T) {
	s := rebindSpec(t, writesHeaderDoc)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := writesCreate(t, eng, "First"); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	got := strings.Join(requestsHeaderNames(reqs[0].header), ",")
	if want := "Accept,Authorization,Content-Type,X-Example-Client"; got != want {
		t.Fatalf("the request holds headers %s, want %s", got, want)
	}
}

// TestWrites_KeyOnlyInAuthorizationOfWrite binds a write with a key to the key in the authorization header and in no other
// part of the request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: The key of a write appears only in the authorization header
func TestWrites_KeyOnlyInAuthorizationOfWrite(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := writesCreate(t, eng, "First"); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	reqs := tr.seen()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want one", len(reqs))
	}
	if !strings.Contains(reqs[0].header.Get("Authorization"), rebindKey) {
		t.Fatalf("the key is not in the authorization header")
	}
	for name, values := range reqs[0].header {
		if name != "Authorization" && strings.Contains(strings.Join(values, ","), rebindKey) {
			t.Fatalf("the key appears in header %s", name)
		}
	}
	if strings.Contains(reqs[0].url.String(), rebindKey) || strings.Contains(string(reqs[0].body), rebindKey) {
		t.Fatalf("the key appears in the path, query, or body")
	}
}

// TestWrites_PathValueLeavingSegmentRefused binds a write path value such as ../admin to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write path value that would leave its segment is refused before any request
func TestWrites_PathValueLeavingSegmentRefused(t *testing.T) {
	s := rebindSpec(t, writesPathDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"id":"doc-1"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := eng.Write(context.Background(), "example-wiki.docs.update", domain.NewArgs(map[string]any{"id": "../admin", "title": "x"}))
	if err == nil || len(tr.seen()) != 0 {
		t.Fatalf("the call is served with %d requests, want a refusal and no request", len(tr.seen()))
	}
}

// TestWrites_UnsetOptionalBodyParamLeavesKeyOut binds an optional body parameter that the caller does not set to a body with
// no key for it.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A body parameter the caller leaves unset leaves its key out of the body
func TestWrites_UnsetOptionalBodyParamLeavesKeyOut(t *testing.T) {
	s := rebindSpec(t, writesOptionalDoc)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := writesCreate(t, eng, "First"); err != nil {
		t.Fatalf("the write fails: %v", err)
	}
	body := writesBody(t, tr)
	if _, present := body["note"]; present {
		t.Fatalf("the body holds a note key %v, want none", body["note"])
	}
}

// TestWrites_HandBuiltPagedWriteRefused binds a hand-built write with a paging rule to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A paged write is refused before any request
func TestWrites_HandBuiltPagedWriteRefused(t *testing.T) {
	s := rebindSpec(t, writesDocOp)
	s.Operations[0].Pagination = &spec.Pagination{Style: "page-all", PageSize: 2, MaxPages: 2}
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	ports := engine.Ports{Credentials: &rebindCredentials{}, Identities: rebindIdentities{}, Clock: &rebindClock{}, Transport: tr}
	eng, err := engine.New(s, rebindSource(t), ports)
	if err == nil {
		_, err = writesCreate(t, eng, "First")
	}
	if err == nil || len(tr.seen()) != 0 {
		t.Fatalf("the paged write is served with %d requests, want a refusal and no request", len(tr.seen()))
	}
}

// TestWrites_UnpagedStyleNoneServedAsUnpagedWrite binds a write whose paging style is none to one unpaged write that succeeds.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write whose paging style is none is served as an unpaged write
func TestWrites_UnpagedStyleNoneServedAsUnpagedWrite(t *testing.T) {
	s := rebindSpec(t, writesNoneDoc)
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := writesCreate(t, eng, "First")
	if err != nil {
		t.Fatalf("the write is refused: %v", err)
	}
	if n := len(tr.seen()); n != 1 || rec.Str("id") != "doc-9" {
		t.Fatalf("%d requests with id %q, want one unpaged write", n, rec.Str("id"))
	}
}

// TestWrites_AdminServedAsWriteAndRefusedAsRead binds an admin operation to a served write and a refused read.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: An admin operation runs as a write and a read of it is still refused
func TestWrites_AdminServedAsWriteAndRefusedAsRead(t *testing.T) {
	s := rebindSpec(t, writesAdminDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"id":"reset-1"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := eng.Write(context.Background(), "example-wiki.admin.reset", domain.NewArgs(nil)); err != nil {
		t.Fatalf("the write path refuses the admin operation: %v", err)
	}
	if _, err := eng.Read(context.Background(), "example-wiki.admin.reset", domain.NewArgs(nil)); !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("the read path error is %v, want the wrong operation class", err)
	}
}

// TestWrites_CatalogRefusalsBeforeRequest binds each call that the catalog refuses to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write that the catalog refuses is refused before any request
func TestWrites_CatalogRefusalsBeforeRequest(t *testing.T) {
	rows := []struct {
		problem string
		op      string
		args    map[string]any
		want    error
	}{
		{problem: "an unknown operation", op: "example-wiki.docs.nope", args: map[string]any{"title": "x"}, want: domain.ErrUnsupportedOperation},
		{problem: "a read operation", op: "example-wiki.docs.one", args: map[string]any{"id": "doc-1"}, want: domain.ErrWrongOperationClass},
		{problem: "an undeclared argument", op: "example-wiki.docs.create", args: map[string]any{"title": "x", "extra": "y"}, want: domain.ErrUnknownArgument},
		{problem: "a missing required argument", op: "example-wiki.docs.create", args: map[string]any{}, want: domain.ErrMissingArgument},
	}
	for _, row := range rows {
		t.Run(row.problem, func(t *testing.T) {
			s := rebindSpec(t, writesDocOp+readsObjectDoc(""))
			tr := rebindReplies(rebindJSON(http.StatusCreated, `{"id":"doc-9"}`))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			_, err := eng.Write(context.Background(), domain.OperationID(row.op), domain.NewArgs(row.args))
			if !errors.Is(err, row.want) || len(tr.seen()) != 0 {
				t.Fatalf("the error is %v with %d requests, want %v and no request", err, len(tr.seen()), row.want)
			}
		})
	}
}

// TestWrites_UnservedConverterRefusedBeforeRequest binds a write whose reply field names a converter the engine does not serve
// to a refusal before any request.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write whose reply needs a converter the engine does not serve is refused before any request
func TestWrites_UnservedConverterRefusedBeforeRequest(t *testing.T) {
	s := rebindSpec(t, writesConvertDoc("markdown-from-html"))
	tr := rebindReplies(rebindJSON(http.StatusCreated, `{"summary":"<p>Hi</p>"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := writesCreate(t, eng, "First")
	if err == nil || len(tr.seen()) != 0 {
		t.Fatalf("the call is served with %d requests, want a refusal and no request", len(tr.seen()))
	}
}

// TestWrites_EditorDataConverterServed binds a write whose reply field names the editor-data converter to a served write.
// @scenario specs/libs/retriever-engine/behaviours/writes-and-outcomes.feature: A write whose reply uses the editor-data converter is served
func TestWrites_EditorDataConverterServed(t *testing.T) {
	s := rebindSpec(t, writesConvertDoc("text-from-editor-data"))
	editor := `{"summary":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hi"}]}]}}`
	tr := rebindReplies(rebindJSON(http.StatusCreated, editor))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := writesCreate(t, eng, "First")
	if err != nil {
		t.Fatalf("the write is refused: %v", err)
	}
	if !strings.Contains(rec.Str("summary"), "Hi") {
		t.Fatalf("the summary is %q, want the plain text of the editor data", rec.Str("summary"))
	}
}
