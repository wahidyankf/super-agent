package engine_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// listTarget is the target of the list operation of the fixture with its default page size.
const listTarget = "/v1/docs?limit=20"

// oneMiB is the number of bytes in one mebibyte, which the reply cap is written in.
const oneMiB = 1 << 20

// TestS3_AReplyOneByteOverTheCapFailsWithReplyTooLarge sends a reply of the default cap plus one byte.
func TestS3_AReplyOneByteOverTheCapFailsWithReplyTooLarge(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", listTarget, replay.Reply{Status: 200, Body: bytes.Repeat([]byte(" "), 10*oneMiB+1)})
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("an over-size reply caused %d requests, want 1", n)
	}
}

// TestS3_TheSpecMayLowerTheReplyCap checks the byte cap that a spec sets. A reply of exactly the cap is read, and one
// byte more fails.
func TestS3_TheSpecMayLowerTheReplyCap(t *testing.T) {
	doc := replaceIn(t, wikiYAML, "  max_pages: 5\noperations:", "  max_pages: 5\n  max_reply_bytes: 64\noperations:")
	eng, stub, _ := newStubEngineFrom(t, loadSpecFrom(t, doc))
	atCap := padded(`{"items":[{"id":"doc-1"}],"next_cursor":null}`, 64)
	stub.Script("GET", listTarget, replay.Reply{Status: 200, Body: atCap})
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
		t.Fatalf("a reply of exactly the cap was refused: %v", err)
	}
	overCap := padded(`{"items":[{"id":"doc-1"}],"next_cursor":null}`, 65)
	stub.Script("GET", listTarget, replay.Reply{Status: 200, Body: overCap})
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("a reply one byte over the spec cap = %v, want ErrReplyTooLarge", err)
	}
}

// TestS3_AStalledReadFailsWithTimedOutAndSendsNoSecondRequest models a stall as the transport's deadline error. The exit
// status of a timeout is 124, and the request is not sent again.
func TestS3_AStalledReadFailsWithTimedOutAndSendsNoSecondRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Fail("GET", listTarget, context.DeadlineExceeded)
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrTimedOut) {
		t.Fatalf("err = %v, want ErrTimedOut", err)
	}
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("exit status = %d, want 124", got)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("a stalled read caused %d requests, want 1", n)
	}
}

// TestS3_ARetryAfterWaitIsTakenUpToTheLimit retries a rate-limited read after each wait and then succeeds.
func TestS3_ARetryAfterWaitIsTakenUpToTheLimit(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script("GET", listTarget,
		replay.Reply{Status: 429, Header: http.Header{"Retry-After": {"1"}}},
		replay.Reply{Status: 429, Header: http.Header{"Retry-After": {"1"}}},
		page(`{"items":[{"id":"doc-1"}],"next_cursor":null}`),
	)
	if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read after two waits: %v", err)
	}
	if n := len(stub.Requests()); n != 3 {
		t.Fatalf("got %d requests, want 3", n)
	}
	if w := clock.waits(); len(w) != 2 || w[0].String() != "1s" || w[1].String() != "1s" {
		t.Fatalf("waits = %v, want two waits of 1s", w)
	}
}

// TestS3_ARetryAfterOverTheLimitIsNotWaitedOut refuses a wait longer than 60 seconds, and sends no second request.
func TestS3_ARetryAfterOverTheLimitIsNotWaitedOut(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script("GET", listTarget, replay.Reply{Status: 429, Header: http.Header{"Retry-After": {"61"}}})
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrRetryAfterTooLong) {
		t.Fatalf("err = %v, want ErrRetryAfterTooLong", err)
	}
	if n := len(stub.Requests()); n != 1 || len(clock.waits()) != 0 {
		t.Fatalf("a long wait caused %d requests and %d waits, want 1 and 0", n, len(clock.waits()))
	}
	if got := engine.ExitStatus(err); got != engine.ExitFailed {
		t.Fatalf("exit status = %d, want 2", got)
	}
}

// TestS3_ARateLimitWithoutAUsableRetryAfterFailsAtOnce checks that a 429 with no wait value is not retried.
func TestS3_ARateLimitWithoutAUsableRetryAfterFailsAtOnce(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", listTarget, replay.Reply{Status: 429})
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("err = %v, want ErrSourceRateLimited", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests, want 1", n)
	}
	if got := engine.ExitStatus(err); got != engine.ExitTimedOut {
		t.Fatalf("exit status = %d, want 124 for a rate limit", got)
	}
}

// TestS3_RetriesAreBoundedAtThreeWaits keeps the source rate limited and checks that the read stops after three waits.
func TestS3_RetriesAreBoundedAtThreeWaits(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script("GET", listTarget, replay.Reply{Status: 429, Header: http.Header{"Retry-After": {"1"}}})
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("err = %v, want ErrSourceRateLimited", err)
	}
	if n := len(stub.Requests()); n != 4 {
		t.Fatalf("got %d requests, want 4 (one try and three retries)", n)
	}
	if n := len(clock.waits()); n != 3 {
		t.Fatalf("got %d waits, want 3", n)
	}
}

// TestS3_AFailedWaitStopsTheRead checks that a cancelled wait ends the read with its own error.
func TestS3_AFailedWaitStopsTheRead(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	clock.sleepErr = context.Canceled
	stub.Script("GET", listTarget, replay.Reply{Status: 429, Header: http.Header{"Retry-After": {"1"}}})
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the wait's error", err)
	}
	if n := len(stub.Requests()); n != 1 {
		t.Fatalf("got %d requests, want 1", n)
	}
}

// TestS4_ARedirectIsNotFollowedAndSendsOneRequest checks that every 3xx reply is refused, and that the stub receives no
// second request to the host the redirect names.
func TestS4_ARedirectIsNotFollowedAndSendsOneRequest(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpec(t))
			stub.Script("GET", listTarget, replay.Reply{
				Status: status,
				Header: http.Header{"Location": {"https://elsewhere.example.test/v1/docs"}},
			})
			_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
			if !errors.Is(err, domain.ErrRedirected) {
				t.Fatalf("err = %v, want ErrRedirected", err)
			}
			if n := len(stub.Requests()); n != 1 {
				t.Fatalf("a redirect caused %d requests, want 1", n)
			}
		})
	}
}

// TestS6_AControlCharacterInAHeaderSlotIsRefusedBeforeAnyRequest refuses a line break in a value that fills a header.
func TestS6_AControlCharacterInAHeaderSlotIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	_, err := eng.Read(context.Background(), "example-wiki.docs.search", domain.NewArgs(map[string]any{
		"note": "first\r\nSet-Cookie: session=1",
	}))
	if !errors.Is(err, engine.ErrInvalidHeader) {
		t.Fatalf("err = %v, want ErrInvalidHeader", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a header with a line break sent %d requests", n)
	}
}

// TestS7_ADotDotValueIsRefusedBeforeAnyRequest checks that a value such as ../admin never reaches a path.
func TestS7_ADotDotValueIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "../admin"}))
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a dot-dot value sent %d requests", n)
	}
}

// TestS7_ALineBreakInAPathValueIsRefusedBeforeAnyRequest checks a value with a line break in a path slot.
func TestS7_ALineBreakInAPathValueIsRefusedBeforeAnyRequest(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": "doc\r\n1"}))
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("a line break in a path sent %d requests", n)
	}
}

// TestS7_ASlashOrPercentInAPathValueIsRefused refuses the separator and the escape character in a path slot.
func TestS7_ASlashOrPercentInAPathValueIsRefused(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	for _, value := range []string{"a/b", "a%2Fb", "%2e%2e", ".", ""} {
		_, err := eng.Read(context.Background(), "example-wiki.docs.get", domain.NewArgs(map[string]any{"id": value}))
		if err == nil {
			t.Fatalf("a path value of %q was accepted", value)
		}
	}
	if n := len(stub.Requests()); n != 0 {
		t.Fatalf("refused path values sent %d requests", n)
	}
}

// TestS7_AQueryValueIsEncodedNeverSpliced checks that a value with reserved characters travels as encoded query text. The
// value holds a tab, which the header rule allows, because the same parameter also fills a header of this operation.
func TestS7_AQueryValueIsEncodedNeverSpliced(t *testing.T) {
	const want = "note=a%26b%3Dc%09d"
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/search?"+want, page(`{"id":"doc-1"}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.search", domain.NewArgs(map[string]any{
		"note": "a&b=c\td",
	})); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if r := stub.Requests()[0]; r.Query != want {
		t.Fatalf("query = %q, want %q", r.Query, want)
	}
}

// TestS9_AReplyNestedPastTheDepthCapFailsAsUnexpected sends a body nested beyond the depth cap, which must fail without
// a panic.
func TestS9_AReplyNestedPastTheDepthCapFailsAsUnexpected(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	deep := `{"items":` + strings.Repeat("[", 70) + strings.Repeat("]", 70) + `}`
	stub.Script("GET", listTarget, page(deep))
	_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestS9_ControlCharactersLeaveTheOutput checks that a bell and a null byte are removed from a mapped text, and that a
// line feed stays.
func TestS9_ControlCharactersLeaveTheOutput(t *testing.T) {
	eng, stub, _ := newStubEngine(t, loadSpec(t))
	stub.Script("GET", listTarget, page(`{"items":[{"id":"doc-1\u0007","title":"First\u0000 document\n"}],"next_cursor":null}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	item := rec.Items()[0]
	if got := item.Str("id"); got != "doc-1" {
		t.Fatalf("id = %q, want the bell removed", got)
	}
	if got := item.Str("title"); got != "First document\n" {
		t.Fatalf("title = %q, want the null removed and the line feed kept", got)
	}
}

// TestS9_AHostileReplyFailsAsUnexpectedAndNeverPanics checks shapes that a list read must refuse.
func TestS9_AHostileReplyFailsAsUnexpectedAndNeverPanics(t *testing.T) {
	bodies := map[string]string{
		"an items value that is not a list": `{"items":"doc-1"}`,
		"an item that is not an object":     `{"items":[1]}`,
		"a reply that is not JSON":          `<html>`,
		"two JSON values in one reply":      `{"items":[]} {"items":[]}`,
		"an empty reply":                    ``,
		"a reply cut off mid-value":         `{"items":[`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			eng, stub, _ := newStubEngine(t, loadSpec(t))
			stub.Script("GET", listTarget, page(body))
			_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}

// TestR1_TheKeyNeverAppearsInAnErrorOrAReply checks that a reply which echoes the key produces an error with no key in
// its text, and that a transport error which carries the key shows it redacted.
func TestR1_TheKeyNeverAppearsInAnErrorOrAReply(t *testing.T) {
	t.Run("a server error that echoes the key", func(t *testing.T) {
		eng, stub, _ := newStubEngine(t, loadSpec(t))
		stub.Script("GET", listTarget, replay.Reply{Status: 500, Body: []byte(`{"error":"` + fixtureKey + `"}`)})
		_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
		if !errors.Is(err, domain.ErrUpstreamFailure) {
			t.Fatalf("err = %v, want ErrUpstreamFailure", err)
		}
		assertNoKey(t, err)
	})
	t.Run("a redirect whose location echoes the key", func(t *testing.T) {
		eng, stub, _ := newStubEngine(t, loadSpec(t))
		stub.Script("GET", listTarget, replay.Reply{Status: 302, Header: http.Header{"Location": {"https://elsewhere.example.test/?k=" + fixtureKey}}})
		_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
		if !errors.Is(err, domain.ErrRedirected) {
			t.Fatalf("err = %v, want ErrRedirected", err)
		}
		assertNoKey(t, err)
	})
	t.Run("a transport error that carries the key", func(t *testing.T) {
		eng, stub, _ := newStubEngine(t, loadSpec(t))
		stub.Fail("GET", listTarget, errors.New("the connection failed with "+fixtureKey))
		_, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
		if !errors.Is(err, domain.ErrUnreachable) {
			t.Fatalf("err = %v, want ErrUnreachable", err)
		}
		assertNoKey(t, err)
		if !strings.Contains(err.Error(), "[REDACTED]") {
			t.Fatalf("the transport text lost its redacted value: %q", err.Error())
		}
	})
	t.Run("a request URL never holds the key", func(t *testing.T) {
		eng, stub, _ := newStubEngine(t, loadSpec(t))
		if _, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil)); err != nil {
			t.Fatalf("Read: %v", err)
		}
		r := stub.Requests()[0]
		if strings.Contains(r.Path+"?"+r.Query, fixtureKey) {
			t.Fatal("the key reached the URL")
		}
	})
}

// assertNoKey fails the test when the error text holds the fixture key.
func assertNoKey(t *testing.T, err error) {
	t.Helper()
	if strings.Contains(err.Error(), fixtureKey) {
		t.Fatalf("the error text holds the key: %q", err.Error())
	}
}

// padded returns the JSON text with trailing spaces, so that it is exactly size bytes long.
func padded(text string, size int) []byte {
	if len(text) > size {
		panic("the test body is longer than its size")
	}
	return []byte(text + strings.Repeat(" ", size-len(text)))
}
