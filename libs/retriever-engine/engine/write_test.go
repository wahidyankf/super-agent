package engine_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// writesYAML is the spec fixture of the write tests. It holds a write that resolves a name, a write with a header slot, an
// admin delete with a spec error map, and a write with a Markdown converter. It names example hosts only.
//
//go:embed testdata/writes.yml
var writesYAML string

// docsPath is the target that the create write and the file write send.
const docsPath = "/v1/docs"

// categoryPath is the target of the lookup that the file write reads for the name Handbook.
const categoryPath = "/v1/categories?name=Handbook"

// createArgs are the arguments of the create write of the wiki fixture.
var createArgs = map[string]any{"title": "First document", "text": "Hello"}

// header returns one header, with its name in canonical form.
func header(name, value string) http.Header {
	h := http.Header{}
	h.Set(name, value)
	return h
}

// created is a canned reply of a create write: 201 with the new ID.
func created(id string) replay.Reply {
	return replay.Reply{Status: http.StatusCreated, Body: []byte(`{"id":"` + id + `"}`)}
}

// writeEngine builds an engine over a replay stub for a spec document, and returns the stub and the fake clock.
func writeEngine(t *testing.T, doc string) (*engine.Engine, *replay.Stub, *fakeClock) {
	t.Helper()
	return newStubEngine(t, loadSpecFrom(t, doc))
}

// requestBody decodes the JSON object that a recorded request carried.
func requestBody(t *testing.T, r replay.Request) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatalf("the body of %s %s is not a JSON object: %v", r.Method, r.Path, err)
	}
	return out
}

// TestS3_AWriteThatTimesOutAfterItLeftIsAnUnknownOutcomeAndIsNotSentAgain checks scenario 1 of pull request 4. A write whose
// stub times out after the request left reports an unknown outcome, with the stable code that the Go connectors give, and
// sends no second request.
func TestS3_AWriteThatTimesOutAfterItLeftIsAnUnknownOutcomeAndIsNotSentAgain(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Fail(http.MethodPost, docsPath, context.DeadlineExceeded)
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("the write caused %d requests, want 1: a write with an unknown outcome is never sent again", n)
	}
	if got := engine.Code("example-wiki", err); got != "retriever.example-wiki.write_outcome_unknown" {
		t.Fatalf("code = %q, want retriever.example-wiki.write_outcome_unknown", got)
	}
	if got := engine.ExitStatus(err); got != engine.ExitFailed {
		t.Fatalf("exit status = %d, want %d: the Go connectors exit 2 for an unknown outcome", got, engine.ExitFailed)
	}
}

// TestAWriteIsNeverSentAgainAfterAnUnknownOutcome checks that every failure after the request left is an unknown outcome,
// and that none of them is retried, so each case sends exactly one request.
func TestAWriteIsNeverSentAgainAfterAnUnknownOutcome(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*replay.Stub)
	}{
		{"a connection that breaks after the send", func(s *replay.Stub) {
			s.Fail(http.MethodPost, docsPath, io.ErrUnexpectedEOF)
		}},
		{"a 500 reply", func(s *replay.Stub) {
			s.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusInternalServerError})
		}},
		{"a 503 reply", func(s *replay.Stub) {
			s.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusServiceUnavailable})
		}},
		{"a 2xx reply that is not JSON", func(s *replay.Stub) {
			s.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusCreated, Body: []byte("not json")})
		}},
		{"a 2xx reply that is a list, not an object", func(s *replay.Stub) {
			s.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusCreated, Body: []byte("[]")})
		}},
		{"a 204 reply with no body", func(s *replay.Stub) {
			s.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusNoContent})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpec(t))
			tc.setup(stub)
			_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
			if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
			}
			if n := len(stub.Requests()); n != 1 {
				t.Fatalf("the write caused %d requests, want 1", n)
			}
		})
	}
}

// TestS9_AWriteReplyNestedPastTheDepthCapIsAnUnknownOutcomeAndNeverPanics checks a hostile reply shape on a write. A reply
// nested past the depth cap is an unknown outcome, and the write is sent once.
func TestS9_AWriteReplyNestedPastTheDepthCapIsAnUnknownOutcomeAndNeverPanics(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	deep := strings.Repeat("[", 70) + strings.Repeat("]", 70)
	stub.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusCreated, Body: []byte(deep)})
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("the write caused %d requests, want 1", n)
	}
}

// TestS3_AWriteReplyOverTheCapIsAnUnknownOutcome checks the size cap on a write. The spec lowers the cap to 10 bytes, and
// the reply holds 14, so the outcome is unknown and the write is not sent again.
func TestS3_AWriteReplyOverTheCapIsAnUnknownOutcome(t *testing.T) {
	doc := replaceIn(t, wikiYAML, "timeout_seconds: 20", "timeout_seconds: 20\n  max_reply_bytes: 10")
	eng, stub, _ := writeEngine(t, doc)
	stub.Script(http.MethodPost, docsPath, created("doc-2"))
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("the write caused %d requests, want 1", n)
	}
}

// TestAWriteThatTheStubAcceptsIsSentOnceWithNoConfirmation checks scenario 2. A write that the stub accepts receives
// exactly one request, no confirmation is asked, and the result maps the declared fields.
func TestAWriteThatTheStubAcceptsIsSentOnceWithNoConfirmation(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	rec, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := rec.Str("id"); got != "doc-2" {
		t.Fatalf("the mapped id = %q, want doc-2", got)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want exactly one", len(reqs))
	}
	r := reqs[0]
	if r.Method != http.MethodPost || r.Path != docsPath {
		t.Fatalf("request = %s %s, want POST %s", r.Method, r.Path, docsPath)
	}
	if got := r.Header.Get("Authorization"); got != "Bearer "+fixtureKey {
		t.Fatalf("authorization = %q, want the bearer key", got)
	}
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
	body := requestBody(t, r)
	if body["title"] != "First document" || body["text"] != "Hello" {
		t.Fatalf("body = %v, want the title and the text", body)
	}
	if n := stub.Unmatched(); n != 0 {
		t.Fatalf("unmatched requests = %d, want 0", n)
	}
}

// TestAWriteLeavesAnUnsetOptionalBodyKeyOut checks that a body parameter the caller does not set leaves its key out.
func TestAWriteLeavesAnUnsetOptionalBodyKeyOut(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	if _, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(map[string]any{"title": "First document"})); err != nil {
		t.Fatalf("Write: %v", err)
	}
	body := requestBody(t, stub.Requests()[0])
	if _, has := body["text"]; has {
		t.Fatalf("body = %v, want no text key", body)
	}
}

// TestS1_ABodyValueIsJSONEncodedAndNeverSpliced checks the typed slots of a body. A value is encoded as JSON, so a quote,
// a line break, and a value that looks like a slot all arrive as the text the caller gave.
func TestS1_ABodyValueIsJSONEncodedAndNeverSpliced(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	text := "line one\n\"two\" {{title}}"
	args := map[string]any{"title": "{{text}}", "text": text}
	if _, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(args)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	r := stub.Requests()[0]
	body := requestBody(t, r)
	if body["text"] != text {
		t.Fatalf("text = %q, want the value as given", body["text"])
	}
	if body["title"] != "{{text}}" {
		t.Fatalf("title = %q, want a value that looks like a slot kept literal", body["title"])
	}
	if strings.Contains(string(r.Body), "\n") {
		t.Fatalf("the body holds a raw line break: the value was spliced, not encoded: %q", r.Body)
	}
}

// TestS6_AHeaderValueWithALineBreakIsRefusedBeforeAnyRequest checks that a header value with a line break adds no header
// line, and that the refusal happens before the request leaves.
func TestS6_AHeaderValueWithALineBreakIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	_, err := eng.Write(context.Background(), "example-wiki.docs.note", domain.NewArgs(map[string]any{"id": "doc-1", "note": "first\r\nX-Evil: yes"}))
	if !errors.Is(err, engine.ErrInvalidHeader) {
		t.Fatalf("err = %v, want ErrInvalidHeader", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused header sent %d requests", n)
	}
	if got := engine.Code("example-wiki", err); got != "retriever.example-wiki.invalid_header" {
		t.Fatalf("code = %q, want retriever.example-wiki.invalid_header", got)
	}
}

// TestS6_AWriteSendsOnlyTheDeclaredHeaders checks that a write carries the declared header, the accept header, the
// content type, and the authorization header, and no other header.
func TestS6_AWriteSendsOnlyTheDeclaredHeaders(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	if _, err := eng.Write(context.Background(), "example-wiki.docs.note", domain.NewArgs(map[string]any{"id": "doc-1", "note": "first"})); err != nil {
		t.Fatalf("Write: %v", err)
	}
	r := stub.Requests()[0]
	if got := r.Header.Get("X-Example-Note"); got != "first" {
		t.Fatalf("X-Example-Note = %q, want first", got)
	}
	allowed := map[string]bool{"Accept": true, "Authorization": true, "Content-Type": true, "X-Example-Note": true}
	for name := range r.Header {
		if !allowed[name] {
			t.Fatalf("the write sent the header %q, which the spec does not declare", name)
		}
	}
}

// TestS7_AWritePathValueIsRefusedBeforeAnyRequest checks that a path value such as ../admin never reaches a path.
func TestS7_AWritePathValueIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	_, err := eng.Write(context.Background(), "example-wiki.docs.note", domain.NewArgs(map[string]any{"id": "../admin", "note": "first"}))
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused path sent %d requests", n)
	}
}

// TestS8_TheKeyTravelsOnlyInTheAuthorizationHeader checks that the key of a write appears in the one header that carries
// it, and in no path, query, body, or other header.
func TestS8_TheKeyTravelsOnlyInTheAuthorizationHeader(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	if _, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	r := stub.Requests()[0]
	if got := r.Header.Get("Authorization"); got != "Bearer "+fixtureKey {
		t.Fatalf("authorization = %q, want the bearer key", got)
	}
	if strings.Contains(r.Path+r.Query+string(r.Body), fixtureKey) {
		t.Fatal("the key appears in the path, the query, or the body")
	}
	for name, values := range r.Header {
		if name != "Authorization" && strings.Contains(strings.Join(values, ","), fixtureKey) {
			t.Fatalf("the key appears in the header %q", name)
		}
	}
}

// TestR1_AWriteErrorReplyNeverEchoesItsBody checks that a 4xx reply to a write is refused without its body in the error.
func TestR1_AWriteErrorReplyNeverEchoesItsBody(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusBadRequest, Body: []byte(`{"message":"` + fixtureKey + ` and an echoed sentence"}`)})
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
	if strings.Contains(err.Error(), fixtureKey) || strings.Contains(err.Error(), "echoed") {
		t.Fatalf("the error echoes the reply: %q", err.Error())
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("the write caused %d requests, want 1", n)
	}
}

// TestS4_AWriteRedirectIsRefusedAfterOneRequest checks that a redirect on a write is refused, and that no second request
// goes to the place it names.
func TestS4_AWriteRedirectIsRefusedAfterOneRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusFound, Header: header("Location", "https://elsewhere.example.test/v1/docs")})
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrRedirected) {
		t.Fatalf("err = %v, want ErrRedirected", err)
	}
	if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatal("a redirect is a known answer, not an unknown outcome")
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("the redirect caused %d requests, want 1", n)
	}
}

// TestAWrite4xxReplyMapsThroughTheSpecErrorMapAndIsKnown checks that each 4xx reply to a write keeps its own sentinel, and
// is never an unknown outcome. The admin delete of the fixture holds the spec error map.
func TestAWrite4xxReplyMapsThroughTheSpecErrorMapAndIsKnown(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, domain.ErrKeyRejected},
		{http.StatusForbidden, domain.ErrAccessDenied},
		{http.StatusBadRequest, domain.ErrBadRequest},
		{http.StatusNotFound, domain.ErrNotFound},
		{http.StatusConflict, domain.ErrDocumentConflict},
		{http.StatusTeapot, domain.ErrUnexpectedResponse},
	}
	for _, c := range cases {
		t.Run(http.StatusText(c.status), func(t *testing.T) {
			eng, stub, _ := writeEngine(t, writesYAML)
			stub.Script(http.MethodDelete, "/v1/docs/doc-1", replay.Reply{Status: c.status, Body: []byte(`{"error":"x"}`)})
			_, err := eng.Write(context.Background(), "example-wiki.docs.remove", domain.NewArgs(map[string]any{"id": "doc-1"}))
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if errors.Is(err, domain.ErrWriteOutcomeUnknown) {
				t.Fatal("a 4xx reply was reported as an unknown outcome")
			}
			reqs := stub.Requests()
			if len(reqs) != 1 || reqs[0].Method != http.MethodDelete {
				t.Fatalf("requests = %+v, want one DELETE", reqs)
			}
		})
	}
}

// TestAWriteIsRetriedAfterA429AsAReadIs checks the retry rule of a write. A 429 reply means the source refused the request
// before it acted, so the write waits the Retry-After time and is sent again, as the Go connectors do.
func TestAWriteIsRetriedAfterA429AsAReadIs(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusTooManyRequests, Header: header("Retry-After", "1")}, created("doc-2"))
	rec, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if err != nil {
		t.Fatalf("Write after one 429: %v", err)
	}
	if got := rec.Str("id"); got != "doc-2" {
		t.Fatalf("id = %q, want doc-2", got)
	}
	if n := len(stub.Requests()); n != 2 {
		t.Fatalf("requests = %d, want 2 (one 429 and one retry)", n)
	}
	if w := clock.waits(); len(w) != 1 || w[0] != time.Second {
		t.Fatalf("waits = %v, want [1s]", w)
	}
}

// TestAWriteStopsAfterThreeRetriesOf429 checks that a write that keeps meeting 429 stops after the same three retries as a
// read, and reports the rate limit with exit status 124.
func TestAWriteStopsAfterThreeRetriesOf429(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script(http.MethodPost, docsPath, replay.Reply{Status: http.StatusTooManyRequests, Header: header("Retry-After", "1")})
	_, err := eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("err = %v, want ErrSourceRateLimited", err)
	}
	if n := len(stub.Requests()); n != 4 {
		t.Fatalf("requests = %d, want 4 (one try and three retries)", n)
	}
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("exit status = %d, want 124", got)
	}
}

// TestAWriteNamedByOneEntryIsSentWithTheResolvedID checks the declared lookup. The name is resolved by one read through
// the engine, and the write carries the ID of the one entry the read matched, not the name.
func TestAWriteNamedByOneEntryIsSentWithTheResolvedID(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	rec, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"}))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := rec.Str("id"); got != "doc-2" {
		t.Fatalf("id = %q, want doc-2", got)
	}
	reqs := stub.Requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want the lookup and the write", len(reqs))
	}
	if reqs[0].Method != http.MethodGet || reqs[0].Query != "name=Handbook" {
		t.Fatalf("the first request = %s ?%s, want the lookup by name", reqs[0].Method, reqs[0].Query)
	}
	if reqs[1].Method != http.MethodPost || reqs[1].Path != docsPath {
		t.Fatalf("the second request = %s %s, want the write", reqs[1].Method, reqs[1].Path)
	}
	body := requestBody(t, reqs[1])
	if body["category"] != float64(7) {
		t.Fatalf("category = %v, want the resolved ID 7", body["category"])
	}
	if strings.Contains(string(reqs[1].Body), "Handbook") {
		t.Fatal("the write carries the name, not only the resolved ID")
	}
}

// TestAWriteWhoseNameMatchesOtherThanOneEntrySendsNoWrite checks scenario 3 and the zero-match case. A name that matches
// two entries, or none, fails with the unresolved-name error, and no write request is sent.
func TestAWriteWhoseNameMatchesOtherThanOneEntrySendsNoWrite(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"two entries match the name", `{"items":[{"id":7,"name":"Handbook"},{"id":8,"name":"Handbook"}]}`},
		{"no entry matches the name", `{"items":[]}`},
		{"an entry differs only in case", `{"items":[{"id":9,"name":"handbook"}]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := writeEngine(t, writesYAML)
			stub.Script(http.MethodGet, categoryPath, replay.Reply{Status: http.StatusOK, Body: []byte(c.body)})
			_, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"}))
			if !errors.Is(err, domain.ErrUnresolvedName) {
				t.Fatalf("err = %v, want ErrUnresolvedName", err)
			}
			if got := engine.Code("example-wiki", err); got != "retriever.example-wiki.unresolved_name" {
				t.Fatalf("code = %q, want retriever.example-wiki.unresolved_name", got)
			}
			for _, r := range stub.Requests() {
				if r.Method != http.MethodGet {
					t.Fatalf("a %s reached the source after an unresolved name", r.Method)
				}
			}
		})
	}
}

// TestAWriteWhoseLookupFailsSendsNoWrite checks that a lookup whose single read fails maps its error as a read does, and
// that no write is sent.
func TestAWriteWhoseLookupFailsSendsNoWrite(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*replay.Stub)
		want  error
	}{
		{"the lookup gets a not found reply", func(s *replay.Stub) {
			s.Script(http.MethodGet, categoryPath, replay.Reply{Status: http.StatusNotFound})
		}, domain.ErrNotFound},
		{"the lookup times out", func(s *replay.Stub) {
			s.Fail(http.MethodGet, categoryPath, context.DeadlineExceeded)
		}, domain.ErrTimedOut},
		{"the lookup is rate limited without a clear", func(s *replay.Stub) {
			s.Script(http.MethodGet, categoryPath, replay.Reply{Status: http.StatusTooManyRequests, Header: header("Retry-After", "1")})
		}, domain.ErrSourceRateLimited},
		{"the lookup reply names no id", func(s *replay.Stub) {
			s.Script(http.MethodGet, categoryPath, replay.Reply{Status: http.StatusOK, Body: []byte(`{"items":[{"name":"Handbook"}]}`)})
		}, domain.ErrUnexpectedResponse},
		{"the lookup reply names the zero id", func(s *replay.Stub) {
			s.Script(http.MethodGet, categoryPath, replay.Reply{Status: http.StatusOK, Body: []byte(`{"items":[{"id":0,"name":"Handbook"}]}`)})
		}, domain.ErrUnexpectedResponse},
		{"the lookup reply names an empty text id", func(s *replay.Stub) {
			s.Script(http.MethodGet, categoryPath, replay.Reply{Status: http.StatusOK, Body: []byte(`{"items":[{"id":"","name":"Handbook"}]}`)})
		}, domain.ErrUnexpectedResponse},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := writeEngine(t, writesYAML)
			c.setup(stub)
			_, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"}))
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			for _, r := range stub.Requests() {
				if r.Method != http.MethodGet {
					t.Fatalf("a %s reached the source after a failed lookup", r.Method)
				}
			}
		})
	}
}

// TestAWriteLookupReadsEveryEntryWhenTheReadHasNoNameParameter checks the lookup of a read that declares no parameter of
// the field's name. The read runs with no name, and the one entry that matches is found by its field.
func TestAWriteLookupReadsEveryEntryWhenTheReadHasNoNameParameter(t *testing.T) {
	doc := replaceIn(t, writesYAML, "      - name: name\n        type: Name\n        in: query", "      - name: label\n        type: Name\n        in: query")
	doc = replaceIn(t, doc, "          name: Handbook\n        response:", "          label: Handbook\n        response:")
	eng, stub, _ := writeEngine(t, doc)
	stub.Script(http.MethodGet, "/v1/categories", replay.Reply{Status: http.StatusOK, Body: []byte(`{"items":[{"id":7,"name":"Handbook"}]}`)})
	if _, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"})); err != nil {
		t.Fatalf("Write: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 2 || reqs[0].Query != "" {
		t.Fatalf("requests = %+v, want the unfiltered lookup then the write", reqs)
	}
}

// TestAWriteNamedByAnEntryWithATextIDSendsThatID checks a lookup whose entry carries a text ID. The write sends the text as
// it came, so the ID that Kiwi or Outline gives as a string resolves as well as a number.
func TestAWriteNamedByAnEntryWithATextIDSendsThatID(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	stub.Script(http.MethodGet, categoryPath, replay.Reply{Status: http.StatusOK, Body: []byte(`{"items":[{"id":"cat-7","name":"Handbook"}]}`)})
	if _, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"})); err != nil {
		t.Fatalf("Write: %v", err)
	}
	body := requestBody(t, stub.Requests()[1])
	if body["category"] != "cat-7" {
		t.Fatalf("category = %v, want the text ID cat-7", body["category"])
	}
}

// TestAWriteWithAnUnsetOptionalLookupSendsNoLookup checks a parameter with a lookup that the caller leaves out. It has no
// name to resolve, so no read runs, and its key is left out of the body.
func TestAWriteWithAnUnsetOptionalLookupSendsNoLookup(t *testing.T) {
	doc := replaceIn(t, writesYAML, "        required: true\n        resolve:", "        required: false\n        resolve:")
	eng, stub, _ := writeEngine(t, doc)
	if _, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document"})); err != nil {
		t.Fatalf("Write: %v", err)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || reqs[0].Method != http.MethodPost {
		t.Fatalf("requests = %+v, want the write alone", reqs)
	}
	if _, has := requestBody(t, reqs[0])["category"]; has {
		t.Fatal("an unset lookup left a category key in the body")
	}
}

// TestAWriteLookupPassesTheCallersArgumentsThatTheReadDeclares checks that the lookup reads with the caller's argument of
// the same name, when the read declares that parameter, and that the write keeps its own argument too.
func TestAWriteLookupPassesTheCallersArgumentsThatTheReadDeclares(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	stub.Script(http.MethodGet, "/v1/categories?folder=Archive&name=Handbook", replay.Reply{Status: http.StatusOK, Body: []byte(`{"items":[{"id":7,"name":"Handbook"}]}`)})
	stub.Script(http.MethodPost, "/v1/docs?folder=Archive", created("doc-2"))
	rec, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook", "folder": "Archive"}))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := rec.Str("id"); got != "doc-2" {
		t.Fatalf("id = %q, want doc-2", got)
	}
	reqs := stub.Requests()
	if len(reqs) != 2 || reqs[0].Query != "folder=Archive&name=Handbook" || reqs[1].Query != "folder=Archive" {
		t.Fatalf("requests = %+v, want the lookup with folder and name, then the write with folder", reqs)
	}
}

// TestAWriteWhoseLookupIsPagedIsRefusedBeforeAnyRequest checks that a lookup over a paged read is refused. A paged read can
// hide a second match past its last page, so the engine never resolves a name over one.
func TestAWriteWhoseLookupIsPagedIsRefusedBeforeAnyRequest(t *testing.T) {
	doc := replaceIn(t, writesYAML,
		"    response:\n      items: /items",
		"    pagination:\n      style: cursor\n      max_pages: 2\n      cursor_param: name\n      next_pointer: /next\n    response:\n      items: /items")
	eng, stub, _ := writeEngine(t, doc)
	_, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused lookup sent %d requests", n)
	}
}

// TestAWriteWithAMarkdownConverterIsRefusedBeforeAnyRequest checks that a write whose reply needs a converter fails closed.
func TestAWriteWithAMarkdownConverterIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	_, err := eng.Write(context.Background(), "example-wiki.docs.revise", domain.NewArgs(map[string]any{"id": "doc-1", "body": "<p>Hello</p>"}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused converter sent %d requests", n)
	}
}

// TestAWritePagedOperationIsRefusedBeforeAnyRequest checks that a write with a paging rule is refused. A write takes one
// request, so a paging rule on it would do nothing, and the engine refuses it instead.
func TestAWritePagedOperationIsRefusedBeforeAnyRequest(t *testing.T) {
	doc := replaceIn(t, writesYAML, "    body:\n      title: \"{{title}}\"",
		"    pagination:\n      style: cursor\n      max_pages: 2\n      cursor_param: folder\n      next_pointer: /next\n    body:\n      title: \"{{title}}\"")
	eng, stub, _ := writeEngine(t, doc)
	_, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"}))
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a refused paged write sent %d requests", n)
	}
}

// TestAWriteWithPagingStyleNoneIsServed checks that a paging style of none, which says there is no paging, is not a
// refusal. The write runs as an unpaged write does.
func TestAWriteWithPagingStyleNoneIsServed(t *testing.T) {
	doc := replaceIn(t, writesYAML, "    body:\n      title: \"{{title}}\"",
		"    pagination:\n      style: none\n    body:\n      title: \"{{title}}\"")
	eng, _, _ := writeEngine(t, doc)
	rec, err := eng.Write(context.Background(), "example-wiki.docs.file", domain.NewArgs(map[string]any{"title": "First document", "category": "Handbook"}))
	if err != nil {
		t.Fatalf("Write with style none: %v", err)
	}
	if got := rec.Str("id"); got != "doc-2" {
		t.Fatalf("id = %q, want doc-2", got)
	}
}

// TestAWriteRefusesWhatTheCatalogRefusesBeforeAnyRequest checks the arguments and the classes that a write refuses: an
// unknown operation, a read, an argument the operation does not declare, and a missing required argument.
func TestAWriteRefusesWhatTheCatalogRefusesBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name string
		op   domain.OperationID
		args map[string]any
		want error
	}{
		{"an operation the spec does not hold", "example-wiki.docs.nothing", nil, domain.ErrUnsupportedOperation},
		{"a read named as a write", "example-wiki.collections.list", map[string]any{}, domain.ErrWrongOperationClass},
		{"an argument the operation does not declare", "example-wiki.docs.create", map[string]any{"title": "x", "page": 2}, domain.ErrUnknownArgument},
		{"a required argument that is missing", "example-wiki.docs.create", map[string]any{"text": "x"}, domain.ErrMissingArgument},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpec(t))
			_, err := eng.Write(context.Background(), c.op, domain.NewArgs(c.args))
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if n := len(stub.Requests()); n != 0 {
				t.Fatalf("a refused call sent %d requests", n)
			}
		})
	}
}

// TestAnAdminOperationIsServedAsAWrite checks that an operation labelled admin runs through the write port, and that a
// read of it is still refused.
func TestAnAdminOperationIsServedAsAWrite(t *testing.T) {
	eng, stub, _ := writeEngine(t, writesYAML)
	rec, err := eng.Write(context.Background(), "example-wiki.docs.remove", domain.NewArgs(map[string]any{"id": "doc-1"}))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := rec.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want doc-1", got)
	}
	reqs := stub.Requests()
	if len(reqs) != 1 || reqs[0].Method != http.MethodDelete || reqs[0].Path != "/v1/docs/doc-1" {
		t.Fatalf("requests = %+v, want one DELETE of the document", reqs)
	}
	if _, err := eng.Read(context.Background(), "example-wiki.docs.remove", domain.NewArgs(map[string]any{"id": "doc-1"})); !errors.Is(err, domain.ErrWrongOperationClass) {
		t.Fatalf("Read of an admin operation = %v, want ErrWrongOperationClass", err)
	}
}

// TestAWriteBodyThatFailsWhileReadIsAnUnknownOutcome checks a reply that breaks while the write's body is read. The write
// has already reached the source, so the outcome is unknown.
func TestAWriteBodyThatFailsWhileReadIsAnUnknownOutcome(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusCreated, Header: http.Header{}, Body: io.NopCloser(brokenBody{})}, nil
	})
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: &fakeCredentials{}, Clock: &fakeClock{}, Transport: transport,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
}

// TestAWriteWithAnInjectedTransportFailingAfterSendIsUnknown checks that a failure of an injected transport is an unknown
// outcome for a write. The injected transport stands for the network, so its failure follows the request.
func TestAWriteWithAnInjectedTransportFailingAfterSendIsUnknown(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	eng, err := engine.New(loadSpec(t), testSource(t), engine.Ports{
		Credentials: &fakeCredentials{}, Clock: &fakeClock{}, Transport: transport,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = eng.Write(context.Background(), "example-wiki.docs.create", domain.NewArgs(createArgs))
	if !errors.Is(err, domain.ErrWriteOutcomeUnknown) {
		t.Fatalf("err = %v, want ErrWriteOutcomeUnknown", err)
	}
}
