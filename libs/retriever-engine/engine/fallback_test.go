package engine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// fallbackDoc is a metadata read with a fallback to a body read, and the body read itself. Both take the same path
// parameter, both are reads of one object, and neither pages, so the fallback loads. The time limit is lowered to 10
// seconds, so the time-limit test proves the two requests share it. It names example hosts only.
const fallbackDoc = `version: 1
kind: example-wiki
summary: Read the metadata of one document, and fall back to its body.
endpoint:
  base_path: /v1
  host_suffix:
    - .example.test
auth:
  type: bearer
  credential_env: EXAMPLE_WIKI_API_KEY
limits:
  timeout_seconds: 10
operations:
  - id: example-wiki.docs.meta
    summary: Read the metadata of one document.
    access: read
    method: GET
    path: /docs/{id}/meta
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        title:
          pointer: /title
        markdown:
          pointer: /markdown
    fallback:
      operation: example-wiki.docs.export
      when:
        empty: markdown
      fill: [markdown]
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            title: First document
            markdown: Full text
        expect:
          title: First document
          markdown: Full text
  - id: example-wiki.docs.export
    summary: Read the body of one document.
    access: read
    method: GET
    path: /docs/{id}/export
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      fields:
        markdown:
          pointer: /markdown
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            markdown: Full text
        expect:
          markdown: Full text
`

const (
	fallbackMetaTarget   = "/v1/docs/doc-1/meta"
	fallbackExportTarget = "/v1/docs/doc-1/export"
)

// fallbackArgs are the arguments of the metadata read.
func fallbackArgs() domain.Args {
	return domain.NewArgs(map[string]any{"id": "doc-1"})
}

// scriptFallbackMeta scripts the first reply of the metadata read.
func scriptFallbackMeta(t *testing.T, stub *replay.Stub, body string) {
	t.Helper()
	stub.Script("GET", fallbackMetaTarget, replay.Reply{Status: 200, Body: []byte(body)})
}

// TestFallbackRunsOnceWhenFieldIsEmpty checks the E8 behaviour: when the mapped markdown field is empty, the engine
// runs the export once and fills markdown with its value, keeping the title of the first reply. The call sends
// exactly two requests, so the fallback runs at most once per call.
func TestFallbackRunsOnceWhenFieldIsEmpty(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, fallbackDoc))
	scriptFallbackMeta(t, stub, `{"title":"First document","markdown":""}`)
	stub.Script("GET", fallbackExportTarget, replay.Reply{Status: 200, Body: []byte(`{"markdown":"# Body"}`)})

	rec, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if err != nil {
		t.Fatalf("the fallback read fails: %v", err)
	}
	if got := rec.Str("title"); got != "First document" {
		t.Fatalf("title = %q; want the title of the first reply", got)
	}
	if got := rec.Str("markdown"); got != "# Body" {
		t.Fatalf("markdown = %q; want the value of the fallback reply", got)
	}
	got := stub.Requests()
	if len(got) != 2 {
		t.Fatalf("got %d requests, want 2", len(got))
	}
	if got[0].Method != "GET" || got[0].Path != "/v1/docs/doc-1/meta" {
		t.Fatalf("first request = %s %s; want the metadata read", got[0].Method, got[0].Path)
	}
	if got[1].Method != "GET" || got[1].Path != "/v1/docs/doc-1/export" {
		t.Fatalf("second request = %s %s; want the export read", got[1].Method, got[1].Path)
	}
}

// TestFallbackSkippedWhenFieldIsPresent checks that a present field sends no second request: the read returns the
// first reply, and the export is never asked. A field that holds a number is present too, so only a missing, null,
// or empty value triggers the fallback.
func TestFallbackSkippedWhenFieldIsPresent(t *testing.T) {
	for _, body := range []string{`{"title":"First document","markdown":"Full text"}`, `{"title":"First document","markdown":5}`} {
		eng, stub, _ := newStubEngine(t, loadSpecFrom(t, fallbackDoc))
		scriptFallbackMeta(t, stub, body)

		if _, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs()); err != nil {
			t.Fatalf("the read fails for %s: %v", body, err)
		}
		if n := len(stub.Requests()); n != 1 {
			t.Fatalf("got %d requests for %s, want 1", n, body)
		}
	}
}

// TestFallbackSharesTheTimeLimit checks that both requests of a fallback read run under one shared time limit: the
// lowered limit of 10 seconds is recorded once for the shared scope and once for each request, and both requests are
// sent. A fallback that opened its own scope would record one limit fewer.
func TestFallbackSharesTheTimeLimit(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpecFrom(t, fallbackDoc))
	scriptFallbackMeta(t, stub, `{"title":"First document","markdown":""}`)
	stub.Script("GET", fallbackExportTarget, replay.Reply{Status: 200, Body: []byte(`{"markdown":"# Body"}`)})

	if _, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs()); err != nil {
		t.Fatalf("the fallback read fails: %v", err)
	}
	if n := len(stub.Requests()); n != 2 {
		t.Fatalf("got %d requests, want 2", n)
	}
	want := 10 * time.Second
	for i, got := range clock.limitsSeen() {
		if got != want {
			t.Fatalf("limit %d = %v; want the shared %v", i, got, want)
		}
	}
	if n := len(clock.limitsSeen()); n != 3 {
		t.Fatalf("got %d time limits, want 3: one shared scope and one per request", n)
	}
}

// TestFallbackFollowsNoRedirect checks that the fallback runs under every safeguard of the engine: when the export
// answers with a redirect, the read fails with the redirect error instead of following it, after exactly two
// requests. No request reaches the host the redirect names.
func TestFallbackFollowsNoRedirect(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, fallbackDoc))
	scriptFallbackMeta(t, stub, `{"title":"First document","markdown":""}`)
	stub.Script("GET", fallbackExportTarget, replay.Reply{Status: 301,
		Header: map[string][]string{"Location": {"https://elsewhere.example.test/v1/docs/doc-1/export"}}})

	_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("err = %v, want ErrRedirected", err)
	}
	if n := len(stub.Requests()); n != 2 {
		t.Fatalf("got %d requests, want 2: the redirect is refused, never followed", n)
	}
}

// TestFallbackRunsOnAMappedStatus checks the status trigger: when the first reply carries a listed status, the
// engine runs the export and fills its fields, even though the first read failed. The output keeps the shape of the
// first read, with the fill fields taken from the fallback and every other field null. A reply whose status is not
// listed returns the first record with no second request.
func TestFallbackRunsOnAMappedStatus(t *testing.T) {
	statusDoc := func(status int) string {
		return replaceIn(t, fallbackDoc, "      when:\n        empty: markdown\n",
			"      when:\n        status: ["+itoa(status)+"]\n")
	}
	cases := []struct {
		name         string
		doc          string
		first        replay.Reply
		wantMarkdown string
		wantTitle    string
		wantRequests int
	}{
		{"a listed error status", statusDoc(404), replay.Reply{Status: 404}, "# Body", "", 2},
		{"a listed success status", statusDoc(200),
			replay.Reply{Status: 200, Body: []byte(`{"title":"First document","markdown":"Full text"}`)},
			"# Body", "", 2},
		{"an unlisted status skips the fallback", statusDoc(404),
			replay.Reply{Status: 200, Body: []byte(`{"title":"First document","markdown":"Full text"}`)},
			"Full text", "First document", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpecFrom(t, c.doc))
			stub.Script("GET", fallbackMetaTarget, c.first)
			stub.Script("GET", fallbackExportTarget,
				replay.Reply{Status: 200, Body: []byte(`{"markdown":"# Body"}`)})

			rec, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
			if err != nil {
				t.Fatalf("the fallback read fails: %v", err)
			}
			if got := rec.Str("markdown"); got != c.wantMarkdown {
				t.Fatalf("markdown = %q; want %q", got, c.wantMarkdown)
			}
			if got := rec.Str("title"); got != c.wantTitle {
				t.Fatalf("title = %q; want %q", got, c.wantTitle)
			}
			if n := len(stub.Requests()); n != c.wantRequests {
				t.Fatalf("got %d requests, want %d", n, c.wantRequests)
			}
		})
	}
}

// TestFallbackFailureFailsTheRead checks that a failed fallback fails the read with the fallback's error: when the
// export answers 404, the read ends with not found, not with the empty field of the first reply, and when the export
// answers a value the mapper refuses, the read ends with that refusal.
func TestFallbackFailureFailsTheRead(t *testing.T) {
	cases := []struct {
		name   string
		export replay.Reply
		want   error
	}{
		{"an error status", replay.Reply{Status: 404}, domain.ErrNotFound},
		{"an unmappable reply", replay.Reply{Status: 200, Body: []byte(`{"markdown":[1]}`)},
			domain.ErrUnexpectedResponse},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpecFrom(t, fallbackDoc))
			scriptFallbackMeta(t, stub, `{"title":"First document","markdown":""}`)
			stub.Script("GET", fallbackExportTarget, c.export)

			_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want the fallback's %v", err, c.want)
			}
			if n := len(stub.Requests()); n != 2 {
				t.Fatalf("got %d requests, want 2", n)
			}
		})
	}
}

// TestFallbackSkippedWhenARequiredFieldIsMissing checks that a first reply the mapper refuses ends the read with the
// mapping failure: when the required markdown field has no value, the read fails before any fallback can fill it,
// after exactly one request.
func TestFallbackSkippedWhenARequiredFieldIsMissing(t *testing.T) {
	doc := replaceIn(t, fallbackDoc, "        markdown:\n          pointer: /markdown\n    fallback:",
		"        markdown:\n          pointer: /markdown\n          required: true\n    fallback:")
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	scriptFallbackMeta(t, stub, `{"title":"First document"}`)

	_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests, want 1", n)
	}
}

// TestFallbackSharesOneReplyBudget checks that both replies draw from the one reply budget of the read: with a small
// reply cap, the export value that no longer fits fails the read with the cap error, after exactly two requests.
func TestFallbackSharesOneReplyBudget(t *testing.T) {
	doc := replaceIn(t, fallbackDoc, "  timeout_seconds: 10\n", "  timeout_seconds: 10\n  max_reply_bytes: 64\n")
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	scriptFallbackMeta(t, stub, `{"title":"abcd","markdown":""}`)
	stub.Script("GET", fallbackExportTarget,
		replay.Reply{Status: 200, Body: []byte(`{"markdown":"0123456789012345678901234567890123456789"}`)})

	_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
	if n := len(stub.Requests()); n != 2 {
		t.Fatalf("got %d requests, want 2", n)
	}
}

// TestFallbackSkippedWhenTheFirstReadFails checks that a failed read is never retried through the fallback: when no
// reply arrives, the read ends with the transport error after exactly one request.
func TestFallbackSkippedWhenTheFirstReadFails(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, fallbackDoc))
	stub.Fail("GET", fallbackMetaTarget, errors.New("connection reset"))

	_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want the transport's ErrUnreachable", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests, want 1: a failed read is not retried", n)
	}
}

// TestFallbackSkippedWhenTheFirstReplyCannotMap checks that a reply the mapper refuses ends the read with the
// mapping error: no fallback can fill a record that never mapped.
func TestFallbackSkippedWhenTheFirstReplyCannotMap(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, fallbackDoc))
	scriptFallbackMeta(t, stub, `{not json`)

	_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests, want 1", n)
	}
}

// TestFallbackTargetWithAConverterFailsClosed checks that a fallback target the engine does not serve fails the
// read before any second mapping: the converter of the target is refused, so raw HTML never passes as Markdown.
func TestFallbackTargetWithAConverterFailsClosed(t *testing.T) {
	doc := replaceIn(t, fallbackDoc, "          pointer: /markdown\n    examples:",
		"          pointer: /markdown\n          convert: markdown-from-html\n    examples:")
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	scriptFallbackMeta(t, stub, `{"title":"First document","markdown":""}`)
	stub.Script("GET", fallbackExportTarget, replay.Reply{Status: 200, Body: []byte(`{"markdown":"<p>Body</p>"}`)})

	_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
}

// TestFallbackTargetWithAMissingValueFailsTheRead checks that the target receives the values of the first call: the
// export requires a revision that the metadata read leaves optional, and when the caller gives none, the read fails
// with the missing argument of the fallback.
func TestFallbackTargetWithAMissingValueFailsTheRead(t *testing.T) {
	withRev := func(doc, path, required string) string {
		return replaceIn(t, doc, "    path: "+path+"\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n",
			"    path: "+path+"\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n      - name: rev\n        type: ID\n        in: query\n        required: "+required+"\n")
	}
	doc := withRev(withRev(fallbackDoc, "/docs/{id}/export", "true"), "/docs/{id}/meta", "false")
	doc = replaceIn(t, doc, "      - request:\n          id: doc-1\n        response:\n          status: 200\n          body:\n            markdown: Full text\n",
		"      - request:\n          id: doc-1\n          rev: doc-9\n        response:\n          status: 200\n          body:\n            markdown: Full text\n")
	eng, stub, _ := newStubEngine(t, loadSpecFrom(t, doc))
	scriptFallbackMeta(t, stub, `{"title":"First document","markdown":""}`)

	_, err := eng.Read(context.Background(), "example-wiki.docs.meta", fallbackArgs())
	if !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("err = %v, want the fallback's ErrMissingArgument", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests, want 1: the fallback never runs without its values", n)
	}
}
