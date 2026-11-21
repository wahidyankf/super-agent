package unit_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// wireFallbackDoc is a metadata read with a fallback to an export read. The export takes the same path parameter, and
// the time limit is lowered to 10 seconds. It names example hosts only.
const wireFallbackDoc = `version: 1
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

// wireDualShapeOps is a read whose fields map two reply shapes: the primary pointer reads the wrapped shape, and the
// fallback pointer reads the unwrapped shape. A 404 carries the mapped reason, unless its body message is exactly the
// declared unavailable message. It names example hosts only.
const wireDualShapeOps = `  - id: example-wiki.docs.get
    summary: Read one document in either reply shape.
    access: read
    method: POST
    read_over_post: true
    path: /docs/get
    params:
      - name: id
        type: ID
        in: body
        required: true
    body:
      id: "{{id}}"
    response:
      fields:
        id:
          pointer: /data/document/id
          fallback_pointer: /data/id
          required: true
        title:
          pointer: /data/document/title
          fallback_pointer: /data/title
      errors:
        "404": not_found
      unavailable_message: Endpoint not found
    examples:
      - request:
          id: doc-1
        response:
          status: 200
          body:
            data:
              id: doc-1
              title: Setup
        expect:
          id: doc-1
`

// wireRun builds the engine for a spec over a round tripper and a clock of its own, and runs one read of the operation.
func wireRun(t *testing.T, s *spec.Spec, rt http.RoundTripper, op domain.OperationID, args domain.Args) (domain.Record, error) {
	t.Helper()
	eng := rebindEngine(t, s, rt, &rebindClock{})
	return eng.Read(context.Background(), op, args)
}

// wireFallbackArgs are the arguments of the metadata read of doc-1.
func wireFallbackArgs() domain.Args {
	return domain.NewArgs(map[string]any{"id": "doc-1"})
}

// wireDualArgs are the arguments of the dual-shape read of doc-1.
func wireDualArgs() domain.Args {
	return domain.NewArgs(map[string]any{"id": "doc-1"})
}

// TestFallbackRunsOnceWhenTheMarkdownFieldIsEmpty runs a metadata read whose markdown is empty, and an export that holds it.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A fallback runs once when the markdown field is empty and fills it with the export value
func TestFallbackRunsOnceWhenTheMarkdownFieldIsEmpty(t *testing.T) {
	model := rebindDoc(t, wireFallbackDoc)
	rt := rebindReplies(
		rebindJSON(200, `{"title":"First document","markdown":""}`),
		rebindJSON(200, `{"markdown":"# Body"}`),
	)
	rec, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if err != nil {
		t.Fatalf("the fallback read fails: %v", err)
	}
	if n := len(rt.seen()); n != 2 {
		t.Fatalf("got %d requests, want exactly two", n)
	}
	if got := rec.Str("markdown"); got != "# Body" {
		t.Fatalf("markdown = %q, want the export value", got)
	}
	if got := rec.Str("title"); got != "First document" {
		t.Fatalf("title = %q, want the title of the first reply kept", got)
	}
}

// TestFallbackIsNotRunWhenTheFilledFieldIsPresent runs the metadata read with a present markdown, and with a number in it.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A fallback is not run when the filled field is present, even when it holds a number
func TestFallbackIsNotRunWhenTheFilledFieldIsPresent(t *testing.T) {
	model := rebindDoc(t, wireFallbackDoc)
	for _, body := range []string{`{"title":"First document","markdown":"Full text"}`, `{"title":"First document","markdown":5}`} {
		rt := rebindReplies(rebindJSON(200, body))
		if _, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs()); err != nil {
			t.Fatalf("the read fails for %s: %v", body, err)
		}
		if n := len(rt.seen()); n != 1 {
			t.Fatalf("got %d requests for %s, want exactly one", n, body)
		}
	}
}

// TestFallbackReadsShareOneTimeLimit records the time limits of a fallback read.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: Both requests of a fallback read share one time limit
func TestFallbackReadsShareOneTimeLimit(t *testing.T) {
	model := rebindDoc(t, wireFallbackDoc)
	rt := rebindReplies(
		rebindJSON(200, `{"title":"First document","markdown":""}`),
		rebindJSON(200, `{"markdown":"# Body"}`),
	)
	clock := &rebindClock{}
	eng := rebindEngine(t, model, rt, clock)
	if _, err := eng.Read(context.Background(), "example-wiki.docs.meta", wireFallbackArgs()); err != nil {
		t.Fatalf("the fallback read fails: %v", err)
	}
	if n := len(rt.seen()); n != 2 {
		t.Fatalf("got %d requests, want both sent", n)
	}
	limits := clock.recordedLimits()
	if len(limits) != 3 {
		t.Fatalf("got %d time limits, want 3: one shared scope and one per request", len(limits))
	}
	for i, d := range limits {
		if d.Seconds() != 10 {
			t.Fatalf("limit %d = %v, want the lowered 10 seconds", i, d)
		}
	}
}

// TestFallbackRedirectIsRefusedAndNothingFollowsIt answers the export with a redirect to another host.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A fallback that is answered with a redirect fails the read and follows nothing
func TestFallbackRedirectIsRefusedAndNothingFollowsIt(t *testing.T) {
	model := rebindDoc(t, wireFallbackDoc)
	rt := rebindReplies(
		rebindJSON(200, `{"title":"First document","markdown":""}`),
		rebindReply{status: http.StatusMovedPermanently, header: http.Header{"Location": {"https://elsewhere.example.test/v1/docs/doc-1/export"}}},
	)
	_, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("err = %v, want ErrRedirected", err)
	}
	seen := rt.seen()
	if len(seen) != 2 {
		t.Fatalf("got %d requests, want exactly two", len(seen))
	}
	for i, req := range seen {
		if req.url.Host != "docs.example.test" {
			t.Fatalf("request %d reached %s, want only docs.example.test", i, req.url.Host)
		}
	}
}

// TestFallbackRunsOnAListedStatusAndKeepsTheFirstShape answers the first read with a listed status, which fails the read.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A fallback runs when the first reply carries a listed status, even though that read failed
func TestFallbackRunsOnAListedStatusAndKeepsTheFirstShape(t *testing.T) {
	model := rebindDoc(t, wireReplace(t, wireFallbackDoc, "      when:\n        empty: markdown\n", "      when:\n        status: [503]\n"))
	rt := rebindReplies(
		rebindJSON(http.StatusServiceUnavailable, `{"title":"First document"}`),
		rebindJSON(200, `{"markdown":"# Body"}`),
	)
	rec, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if err != nil {
		t.Fatalf("the fallback read fails: %v", err)
	}
	if got := rec.Str("markdown"); got != "# Body" {
		t.Fatalf("markdown = %q, want the export value", got)
	}
	if v, present := rec.Get("title"); !present || v != nil {
		t.Fatalf("title = %v (present %v), want null in the shape of the first read", v, present)
	}
}

// TestFailedFallbackFailsWithItsOwnError answers the export with not found.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A failed fallback fails the read with its own error and not with the empty field of the first reply
func TestFailedFallbackFailsWithItsOwnError(t *testing.T) {
	model := rebindDoc(t, wireFallbackDoc)
	rt := rebindReplies(
		rebindJSON(200, `{"title":"First document","markdown":""}`),
		rebindJSON(http.StatusNotFound, `{}`),
	)
	_, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want the not-found error of the export", err)
	}
}

// TestRequiredMissingFieldEndsTheReadBeforeAnyFallback drops the required markdown field from the first reply.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A required field that is missing in the first reply ends the read before any fallback runs
func TestRequiredMissingFieldEndsTheReadBeforeAnyFallback(t *testing.T) {
	model := rebindDoc(t, wireReplace(t, wireFallbackDoc, "        markdown:\n          pointer: /markdown\n    fallback:",
		"        markdown:\n          pointer: /markdown\n          required: true\n    fallback:"))
	rt := rebindReplies(rebindJSON(200, `{"title":"First document"}`))
	_, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want the mapping failure", err)
	}
	if n := len(rt.seen()); n != 1 {
		t.Fatalf("got %d requests, want exactly one", n)
	}
}

// TestBothRepliesOfAFallbackDrawFromOneBudget sets a reply cap that the second reply overflows once the first is counted.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: Both replies of a fallback read draw from one reply budget
func TestBothRepliesOfAFallbackDrawFromOneBudget(t *testing.T) {
	model := rebindDoc(t, wireReplace(t, wireFallbackDoc, "timeout_seconds: 10", "timeout_seconds: 10\n  max_reply_bytes: 100"))
	title := strings.Repeat("t", 50)
	export := strings.Repeat("e", 60)
	rt := rebindReplies(
		rebindJSON(200, `{"title":"`+title+`","markdown":""}`),
		rebindJSON(200, `{"markdown":"`+export+`"}`),
	)
	_, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want the reply-size error", err)
	}
	if n := len(rt.seen()); n != 2 {
		t.Fatalf("got %d requests, want exactly two", n)
	}
}

// TestFailedFirstReadIsNeverRetriedThroughTheFallback gives the first request no reply.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A failed first read is never retried through the fallback
func TestFailedFirstReadIsNeverRetriedThroughTheFallback(t *testing.T) {
	model := rebindDoc(t, wireFallbackDoc)
	rt := rebindReplies(rebindReply{err: errors.New("the connection was refused by the fixture")})
	_, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("err = %v, want the transport error", err)
	}
	if n := len(rt.seen()); n != 1 {
		t.Fatalf("got %d requests, want exactly one", n)
	}
}

// TestUnmappableFirstReplyEndsTheReadWithoutAFallback answers the first read with a body that is not JSON.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A first reply the mapper refuses ends the read with the mapping error and no fallback runs
func TestUnmappableFirstReplyEndsTheReadWithoutAFallback(t *testing.T) {
	model := rebindDoc(t, wireFallbackDoc)
	rt := rebindReplies(rebindJSON(200, `not json`))
	_, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want the mapping error", err)
	}
	if n := len(rt.seen()); n != 1 {
		t.Fatalf("got %d requests, want no fallback request", n)
	}
}

// TestFallbackTargetWithAnUnservedConverterIsUnsupported gives the export a converter that the engine does not serve.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A fallback target that needs a converter the engine does not serve fails before any second mapping
func TestFallbackTargetWithAnUnservedConverterIsUnsupported(t *testing.T) {
	model := rebindDoc(t, wireReplace(t, wireFallbackDoc, "          pointer: /markdown\n    examples:",
		"          pointer: /markdown\n          convert: markdown-from-html\n    examples:"))
	rt := rebindReplies(
		rebindJSON(200, `{"title":"First document","markdown":""}`),
		rebindJSON(200, `{"markdown":"<p>raw markup</p>"}`),
	)
	rec, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if rec.Str("markdown") != "" {
		t.Fatal("raw markup passed as Markdown")
	}
}

// TestFallbackTargetNeedingAMissingArgumentFails gives the export a required revision that the caller does not give. The
// metadata read declares the revision too, as an optional parameter, because a fallback target may not declare a parameter
// that the first read does not declare.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A fallback target that needs a value the caller does not give fails with the missing argument
func TestFallbackTargetNeedingAMissingArgumentFails(t *testing.T) {
	doc := wireReplace(t, wireFallbackDoc, "    path: /docs/{id}/meta\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n",
		"    path: /docs/{id}/meta\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n"+
			"      - name: revision\n        type: ID\n        in: query\n        required: false\n")
	doc = wireReplace(t, doc, "    path: /docs/{id}/export\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n",
		"    path: /docs/{id}/export\n    params:\n      - name: id\n        type: ID\n        in: path\n        required: true\n"+
			"      - name: revision\n        type: ID\n        in: query\n        required: true\n")
	model := rebindDoc(t, doc)
	rt := rebindReplies(rebindJSON(200, `{"title":"First document","markdown":""}`))
	_, err := wireRun(t, model, rt, "example-wiki.docs.meta", wireFallbackArgs())
	if !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("err = %v, want the missing-argument error", err)
	}
}

// TestFallbackPointerMapsTheFallbackShapeOfAReply answers with the unwrapped shape only.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A reply that lacks the primary pointer but holds the fallback shape maps the fallback shape
func TestFallbackPointerMapsTheFallbackShapeOfAReply(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireDualShapeOps)
	rt := rebindReplies(rebindJSON(200, `{"data":{"id":"doc-1","title":"Setup"}}`))
	rec, err := wireRun(t, model, rt, "example-wiki.docs.get", wireDualArgs())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want the fallback pointer value", got)
	}
	if got := rec.Str("title"); got != "Setup" {
		t.Fatalf("title = %q, want the fallback pointer value of the other field", got)
	}
}

// TestPrimaryShapeLeavesTheFallbacksUnused answers with the wrapped shape, and holds fallback values that would differ.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A reply that holds the primary shape maps the primary pointers and leaves the fallbacks unused
func TestPrimaryShapeLeavesTheFallbacksUnused(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireDualShapeOps)
	rt := rebindReplies(rebindJSON(200, `{"data":{"document":{"id":"doc-1","title":"Primary"},"id":"other","title":"Fallback"}}`))
	rec, err := wireRun(t, model, rt, "example-wiki.docs.get", wireDualArgs())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := rec.Str("title"); got != "Primary" {
		t.Fatalf("title = %q, want the primary pointer value", got)
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want the primary pointer value", got)
	}
}

// TestPrimaryNullWinsOverTheFallback answers with an explicit null at the primary pointer, with a fallback value present.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A primary pointer that finds an explicit null wins over the fallback
func TestPrimaryNullWinsOverTheFallback(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireDualShapeOps)
	rt := rebindReplies(rebindJSON(200, `{"data":{"document":{"id":"doc-1","title":null},"title":"Fallback"}}`))
	rec, err := wireRun(t, model, rt, "example-wiki.docs.get", wireDualArgs())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if v, present := rec.Get("title"); !present || v != nil {
		t.Fatalf("title = %v (present %v), want the explicit null", v, present)
	}
}

// TestUnavailableMessageOn404FailsAsOperationUnavailable answers a 404 with the declared message.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A 404 with the declared unavailable message fails as operation unavailable
func TestUnavailableMessageOn404FailsAsOperationUnavailable(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireDualShapeOps)
	rt := rebindReplies(rebindJSON(http.StatusNotFound, `{"message":"Endpoint not found"}`))
	_, err := wireRun(t, model, rt, "example-wiki.docs.get", wireDualArgs())
	var unavailable domain.OperationUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("err = %v, want OperationUnavailableError", err)
	}
}

// TestNotFoundWithoutTheMessageKeepsItsReason answers a 404 with another message.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A 404 without the declared message keeps its mapped not-found reason
func TestNotFoundWithoutTheMessageKeepsItsReason(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireDualShapeOps)
	rt := rebindReplies(rebindJSON(http.StatusNotFound, `{"message":"No document with this ID"}`))
	_, err := wireRun(t, model, rt, "example-wiki.docs.get", wireDualArgs())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// TestUnavailableMessageOn401KeepsTheStatusError answers a 401 that carries the declared message.
//
// @scenario specs/libs/retriever-engine/behaviours/fallbacks-and-shapes.feature: A 401 that carries the declared message keeps the error of its own status
func TestUnavailableMessageOn401KeepsTheStatusError(t *testing.T) {
	model := rebindDoc(t, rebindHeader+wireDualShapeOps)
	rt := rebindReplies(rebindJSON(http.StatusUnauthorized, `{"message":"Endpoint not found"}`))
	_, err := wireRun(t, model, rt, "example-wiki.docs.get", wireDualArgs())
	if !errors.Is(err, domain.ErrKeyRejected) {
		t.Fatalf("err = %v, want ErrKeyRejected", err)
	}
	var unavailable domain.OperationUnavailableError
	if errors.As(err, &unavailable) {
		t.Fatal("a 401 was reported as operation unavailable")
	}
}
