package unit_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The reply safeguards of these bindings read each canned reply of the in-memory transport. Every test builds its reply in
// memory, so the size and shape limits are exercised without any socket or file.

// replyEchoDoc is a single-document read whose reply must echo the id that the caller asked for.
const replyEchoDoc = `  - id: example-wiki.docs.one
    summary: Read one document with a key echo.
    access: read
    method: GET
    path: /docs/{id}
    params:
      - name: id
        type: ID
        in: path
        required: true
    response:
      echo:
        param: id
        pointer: /id
      fields:
        title:
          pointer: /title
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            id: doc-1
            title: First document
        expect:
          title: First document
`

// replyCapHeader is the header of the bindings with a lowered reply cap of 256 bytes.
var replyCapHeader = strings.Replace(rebindHeader, "limits:\n  timeout_seconds: 20\n", "limits:\n  timeout_seconds: 20\n  max_reply_bytes: 256\n", 1)

// replyTitleJSON returns a reply of exactly size bytes whose title is padded with letters.
func replyTitleJSON(t *testing.T, size int) string {
	t.Helper()
	const prefix = `{"title":"`
	const suffix = `"}`
	pad := size - len(prefix) - len(suffix)
	if pad < 0 {
		t.Fatalf("the reply of %d bytes is too small", size)
	}
	body := prefix + strings.Repeat("a", pad) + suffix
	if len(body) != size {
		t.Fatalf("the reply is %d bytes, want %d", len(body), size)
	}
	return body
}

// TestReplySafeguards_ReplyOneByteOverDefaultCapRefused binds a reply one byte over the default 10 MiB cap to the reply-size
// error.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A reply one byte over the default 10 MiB cap fails with the reply-size error
func TestReplySafeguards_ReplyOneByteOverDefaultCapRefused(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(rebindJSON(http.StatusOK, strings.Repeat(" ", 10<<20+1)))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("the error is %v, want the reply-size error", err)
	}
}

// TestReplySafeguards_LoweredCapReadsExactAndRefusesOneMore binds a spec that lowers the reply cap to a reply of exactly the
// cap, which is read, and to a reply one byte more, which fails with the reply-size error.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A spec that lowers the reply cap reads a reply of exactly the cap and refuses one byte more
func TestReplySafeguards_LoweredCapReadsExactAndRefusesOneMore(t *testing.T) {
	s := rebindDoc(t, replyCapHeader+readsObjectDoc(""))
	tr := rebindReplies(
		rebindJSON(http.StatusOK, replyTitleJSON(t, 256)),
		rebindJSON(http.StatusOK, replyTitleJSON(t, 257)),
	)
	eng := rebindEngine(t, s, tr, &rebindClock{})
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); err != nil {
		t.Fatalf("the reply of exactly the cap is refused: %v", err)
	}
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("the reply one byte over the cap gives %v, want the reply-size error", err)
	}
}

// TestReplySafeguards_TooDeepReplyUnexpectedNoPanic binds a reply nested past the depth cap to an unexpected response, with no
// panic.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A reply nested past the depth cap fails as unexpected and does not panic
func TestReplySafeguards_TooDeepReplyUnexpectedNoPanic(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	deep := `{"title":"First document","deep":` + strings.Repeat("[", 70) + strings.Repeat("]", 70) + `}`
	tr := rebindReplies(rebindJSON(http.StatusOK, deep))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("the error is %v, want an unexpected response", err)
	}
}

// TestReplySafeguards_TooManyNodesUnexpected binds a reply with more nodes than the token cap to an unexpected response.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A reply with more nodes than the token cap fails as unexpected
func TestReplySafeguards_TooManyNodesUnexpected(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	wide := `{"title":"First document","items":[` + strings.Repeat("0,", 1<<20) + `0]}`
	tr := rebindReplies(rebindJSON(http.StatusOK, wide))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("the error is %v, want an unexpected response", err)
	}
}

// TestReplySafeguards_NotOneWholeJSONValueUnexpected binds a reply that is not one whole JSON value to an unexpected response.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A reply that is not one whole JSON value fails as unexpected
func TestReplySafeguards_NotOneWholeJSONValueUnexpected(t *testing.T) {
	rows := []struct{ shape, body string }{
		{shape: "two JSON values one after another", body: `{"title":"a"}{"title":"b"}`},
		{shape: "text that is not JSON", body: `not json at all`},
	}
	for _, row := range rows {
		t.Run(row.shape, func(t *testing.T) {
			s := rebindSpec(t, readsObjectDoc(""))
			tr := rebindReplies(rebindJSON(http.StatusOK, row.body))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("the error is %v, want an unexpected response", err)
			}
		})
	}
}

// TestReplySafeguards_ControlCharactersStrippedExceptLineFeed binds a mapped text with a bell, a null byte, and a line feed to
// a text with no bell and no null byte, and the line feed kept.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A mapped text keeps its line feeds and loses its bell and null characters
func TestReplySafeguards_ControlCharactersStrippedExceptLineFeed(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"a\u0007b\u0000c\nd"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	title := rec.Str("title")
	if strings.ContainsRune(title, '\a') || strings.ContainsRune(title, 0) {
		t.Fatalf("the mapped text %q keeps a bell or a null byte", title)
	}
	if !strings.Contains(title, "\n") {
		t.Fatalf("the mapped text %q lost its line feed", title)
	}
}

// TestReplySafeguards_HostileListShapeUnexpected binds a hostile reply shape in a list read to an unexpected response, with no
// panic.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A hostile reply shape in a list read fails as unexpected and never panics
func TestReplySafeguards_HostileListShapeUnexpected(t *testing.T) {
	s := rebindSpec(t, readsListDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"items":{"id":"doc-1"},"next_cursor":null}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.list", nil)
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("the error is %v, want an unexpected response", err)
	}
}

// TestReplySafeguards_MappedTextOverOutputBudgetRefused binds two fields that map the same six-MiB text, which together pass
// the output budget of one reply, to the reply-size error.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: Mapped text that would pass the output budget fails the read
func TestReplySafeguards_MappedTextOverOutputBudgetRefused(t *testing.T) {
	doc := readsObjectDoc("        body:\n          pointer: /body\n        second:\n          pointer: /body\n")
	s := rebindSpec(t, doc)
	big := strings.Repeat("a", 6<<20)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","body":"`+big+`"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("the error is %v, want the reply-size error", err)
	}
}

// TestReplySafeguards_NumberDecodedWithoutLoss binds a number with many digits to the exact number in the mapped field.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A number in a reply is decoded without loss
func TestReplySafeguards_NumberDecodedWithoutLoss(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc("        count:\n          pointer: /count\n"))
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","count":12345678901234567890123}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got, _ := rec.Get("count"); fmt.Sprint(got) != "12345678901234567890123" {
		t.Fatalf("the field holds %v, want the exact number", got)
	}
}

// TestReplySafeguards_KeyKeptOutOfErrorAndReply binds a transport error that carries the API key to an error with no key, and a
// reply that echoes the key to a record with no key.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A reply that echoes the key of a failed read keeps the key out of the error and the reply
func TestReplySafeguards_KeyKeptOutOfErrorAndReply(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	failing := rebindReplies(rebindReply{err: errors.New("the connection to the source failed with " + rebindKey)})
	eng := rebindEngine(t, s, failing, &rebindClock{})
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err == nil || strings.Contains(err.Error(), rebindKey) {
		t.Fatalf("the error is %v, want a failure with no key in it", err)
	}
	echoing := rebindReplies(rebindJSON(http.StatusOK, `{"title":"First document","echo":"`+rebindKey+`"}`))
	eng = rebindEngine(t, s, echoing, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if strings.Contains(fmt.Sprint(rec.Fields()), rebindKey) {
		t.Fatalf("the record holds the key")
	}
}

// TestReplySafeguards_MatchingKeyEchoServesRead binds a key echo that equals the request argument to the mapped record.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A key echo that matches the request key serves the read
func TestReplySafeguards_MatchingKeyEchoServesRead(t *testing.T) {
	s := rebindSpec(t, replyEchoDoc)
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"id":"doc-1","title":"First document"}`))
	eng := rebindEngine(t, s, tr, &rebindClock{})
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if rec.Str("title") != "First document" {
		t.Fatalf("the record holds title %q, want the mapped record", rec.Str("title"))
	}
}

// TestReplySafeguards_MismatchedKeyEchoUnexpected binds a key echo that does not match a text value at its pointer to an
// unexpected response naming the echo path, with neither key shown.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A key echo that does not match a text value at its pointer fails as unexpected
func TestReplySafeguards_MismatchedKeyEchoUnexpected(t *testing.T) {
	rows := []struct{ reply, body string }{
		{reply: "holds a different key", body: `{"id":"doc-2","title":"First document"}`},
		{reply: "holds a number", body: `{"id":5,"title":"First document"}`},
		{reply: "holds no value", body: `{"title":"First document"}`},
	}
	for _, row := range rows {
		t.Run(row.reply, func(t *testing.T) {
			s := rebindSpec(t, replyEchoDoc)
			tr := rebindReplies(rebindJSON(http.StatusOK, row.body))
			eng := rebindEngine(t, s, tr, &rebindClock{})
			_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
			if !errors.Is(err, domain.ErrUnexpectedResponse) || !strings.Contains(err.Error(), "/id") {
				t.Fatalf("the error is %v, want an unexpected response naming /id", err)
			}
			if strings.Contains(err.Error(), "doc-1") || strings.Contains(err.Error(), "doc-2") {
				t.Fatalf("the error shows a key: %v", err)
			}
		})
	}
}

// TestReplySafeguards_KeyEchoOnListRefusedAtBuild binds a hand-built model with a key echo on a list response to a refusal
// when the engine is built, with no request sent.
// @scenario specs/libs/retriever-engine/behaviours/reply-safeguards-and-echo.feature: A key echo on a list read is refused before any request
func TestReplySafeguards_KeyEchoOnListRefusedAtBuild(t *testing.T) {
	s := rebindSpec(t, readsListDoc)
	s.Operations[0].Response.Echo = &spec.Echo{Param: "limit", Pointer: "/id"}
	tr := rebindReplies(rebindJSON(http.StatusOK, `{"items":[],"next_cursor":null}`))
	ports := engine.Ports{Credentials: &rebindCredentials{}, Identities: rebindIdentities{}, Clock: &rebindClock{}, Transport: tr}
	_, err := engine.New(s, rebindSource(t), ports)
	if err == nil || len(tr.seen()) != 0 {
		t.Fatalf("the engine is built with %d requests, want a refusal at build and no request", len(tr.seen()))
	}
}
