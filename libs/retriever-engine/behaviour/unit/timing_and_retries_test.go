package unit_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
)

// The timing of these bindings uses the injected clock and sleeper only. Every wait is recorded and returns at once, and no
// test reads the real clock, starts a timer, or sleeps.

// timingErrWait is the error that a cancelled wait returns.
var timingErrWait = errors.New("the wait was cancelled")

// timingRetryAfter returns a 429 reply with the given Retry-After value; an empty value sends no Retry-After header.
func timingRetryAfter(value string) rebindReply {
	rep := rebindReply{status: http.StatusTooManyRequests}
	if value != "" {
		rep.header = http.Header{"Retry-After": {value}}
	}
	return rep
}

// timingTitleOK is the success reply of the single-document read.
func timingTitleOK() rebindReply {
	return rebindJSON(http.StatusOK, `{"title":"First document"}`)
}

// TestTiming_RateLimitedReadWaitsRetryAfterThenSucceeds binds a 429 with a Retry-After of one second, followed by a success,
// to one wait of that second before the second request and the mapped result.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A rate-limited read waits the Retry-After time and then succeeds
func TestTiming_RateLimitedReadWaitsRetryAfterThenSucceeds(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(timingRetryAfter("1"), timingTitleOK())
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	rec, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[1s]" {
		t.Fatalf("the waits are %s, want one wait of the Retry-After time", got)
	}
	if len(tr.seen()) != 2 || rec.Str("title") != "First document" {
		t.Fatalf("%d requests and title %q, want two requests and the mapped success", len(tr.seen()), rec.Str("title"))
	}
}

// TestTiming_RetryAfterOverSixtySecondsNotWaited binds a Retry-After of more than 60 seconds to the retry-after-too-long
// failure, with exit status 2, no wait, and no second request.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A Retry-After wait over 60 seconds is not waited out and no second request is sent
func TestTiming_RetryAfterOverSixtySecondsNotWaited(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(timingRetryAfter("61"), timingTitleOK())
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrRetryAfterTooLong) || engine.ExitStatus(err) != 2 {
		t.Fatalf("the error is %v with exit status %d, want the retry-after-too-long error and exit status 2", err, engine.ExitStatus(err))
	}
	if n := len(tr.seen()); n != 1 || fmt.Sprint(clock.recordedWaits()) != "[]" {
		t.Fatalf("%d requests and waits %v, want one request and no wait", n, clock.recordedWaits())
	}
}

// TestTiming_RateLimitWithoutUsableWaitNotRetried binds a 429 with no usable Retry-After to a rate-limit failure at once,
// with exactly one request.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A 429 reply with no usable Retry-After is not retried
func TestTiming_RateLimitWithoutUsableWaitNotRetried(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(timingRetryAfter(""), timingTitleOK())
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrSourceRateLimited) {
		t.Fatalf("the error is %v, want the source rate-limit error", err)
	}
	if n := len(tr.seen()); n != 1 || fmt.Sprint(clock.recordedWaits()) != "[]" {
		t.Fatalf("%d requests and waits %v, want one request and no wait", n, clock.recordedWaits())
	}
}

// TestTiming_RateLimitKeptAfterThreeWaits binds a source that keeps answering 429 with a usable wait to a failure after three
// waits, with exit status 124.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A source that keeps rate-limiting stops the read after three waits
func TestTiming_RateLimitKeptAfterThreeWaits(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(timingRetryAfter("1"))
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, domain.ErrSourceRateLimited) || engine.ExitStatus(err) != engine.ExitTimedOut {
		t.Fatalf("the error is %v with exit status %d, want the rate-limit error and exit status 124", err, engine.ExitStatus(err))
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[1s 1s 1s]" || len(tr.seen()) != 4 {
		t.Fatalf("the waits are %s with %d requests, want three waits and four requests", got, len(tr.seen()))
	}
}

// TestTiming_CancelledWaitEndsReadWithItsError binds a retry wait that is cancelled to the error of that wait.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A wait that is cancelled ends the read with the wait's own error
func TestTiming_CancelledWaitEndsReadWithItsError(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(timingRetryAfter("1"), timingTitleOK())
	clock := &rebindClock{sleepErr: timingErrWait}
	eng := rebindEngine(t, s, tr, clock)
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if !errors.Is(err, timingErrWait) {
		t.Fatalf("the error is %v, want the error of the cancelled wait", err)
	}
}

// TestTiming_StalledReadTimesOutWithoutSecondRequest binds a stalled read under a lowered time limit to a timeout with exit
// status 124 and exactly one request.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A stalled read stops at its time limit with the timeout exit status and sends no second request
func TestTiming_StalledReadTimesOutWithoutSecondRequest(t *testing.T) {
	s := rebindDoc(t, strings.Replace(rebindHeader, "timeout_seconds: 20", "timeout_seconds: 1", 1)+readsObjectDoc(""))
	tr := rebindReplies(rebindReply{stall: true})
	clock := &rebindClock{expired: true}
	eng := rebindEngine(t, s, tr, clock)
	_, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"})
	if engine.ExitStatus(err) != engine.ExitTimedOut {
		t.Fatalf("the error is %v with exit status %d, want the timeout exit status 124", err, engine.ExitStatus(err))
	}
	if n := len(tr.seen()); n != 1 {
		t.Fatalf("%d requests, want exactly one", n)
	}
	if got := fmt.Sprint(clock.recordedLimits()); got != "[1s]" {
		t.Fatalf("the time limits are %s, want the lowered limit of one second", got)
	}
}

// TestTiming_ShortRetryAfterWaitsMinimumDelay binds two short Retry-After waits to the 100 ms minimum before each retry.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A retried single read waits the 100 ms minimum delay after a short Retry-After and keeps the spacing
func TestTiming_ShortRetryAfterWaitsMinimumDelay(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(timingRetryAfter("0"), timingRetryAfter("0"), timingTitleOK())
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[100ms 100ms]" {
		t.Fatalf("the waits are %s, want two waits of 100 ms", got)
	}
}

// TestTiming_LongRetryAfterWaitKeptAsAsked binds a retry wait longer than the minimum delay to the Retry-After time, with
// nothing added.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A retry wait longer than the minimum delay is kept as the reply asks
func TestTiming_LongRetryAfterWaitKeptAsAsked(t *testing.T) {
	s := rebindSpec(t, readsObjectDoc(""))
	tr := rebindReplies(timingRetryAfter("2"), timingTitleOK())
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	if _, err := pagingRead(eng, "example-wiki.docs.one", map[string]any{"id": "doc-1"}); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[2s]" {
		t.Fatalf("the waits are %s, want the Retry-After time alone", got)
	}
}

// TestTiming_PagesWaitMinimumDelayBeforeEachPageAfterFirst binds each paging style to one 100 ms wait before every page after
// the first, and no other wait.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: The engine waits the 100 ms minimum delay before every <paging> page after the first
func TestTiming_PagesWaitMinimumDelayBeforeEachPageAfterFirst(t *testing.T) {
	rows := []struct {
		paging, doc, op string
		args            map[string]any
		replies         []rebindReply
	}{
		{
			paging: "offset", doc: pagingOffsetOp, op: "example-wiki.docs.page", args: map[string]any{"limit": 2},
			replies: []rebindReply{
				pagingPage(pagingDocs(1, 2), nil),
				pagingPage(pagingDocs(3, 4), nil),
				pagingPage(pagingDocs(5, 5), nil),
			},
		},
		{
			paging: "cursor", doc: pagingCursorOp, op: "example-wiki.docs.list",
			replies: []rebindReply{
				pagingPage([]string{"doc-1"}, map[string]any{"next_cursor": "c2"}),
				pagingPage([]string{"doc-2"}, map[string]any{"next_cursor": "c3"}),
				pagingPage([]string{"doc-3"}, map[string]any{"next_cursor": nil}),
			},
		},
		{
			paging: "page-all", doc: pagingPageAllOp, op: "example-wiki.docs.all",
			replies: []rebindReply{
				pagingPage(pagingDocs(1, 2), nil),
				pagingPage(pagingDocs(3, 4), nil),
				pagingPage(pagingDocs(5, 5), nil),
			},
		},
	}
	for _, row := range rows {
		t.Run(row.paging, func(t *testing.T) {
			s := rebindSpec(t, row.doc)
			tr := rebindReplies(row.replies...)
			clock := &rebindClock{}
			eng := rebindEngine(t, s, tr, clock)
			if _, err := pagingRead(eng, row.op, row.args); err != nil {
				t.Fatalf("the read fails: %v", err)
			}
			waits := clock.recordedWaits()
			if len(waits) != len(tr.seen())-1 {
				t.Fatalf("%d waits for %d pages, want one wait before each page after the first", len(waits), len(tr.seen()))
			}
			for i, w := range waits {
				if w != 100*time.Millisecond {
					t.Fatalf("wait %d is %v, want 100ms", i, w)
				}
			}
		})
	}
}

// TestTiming_PageAllShortRetryTopsUpToMinimum binds a short 429 wait on a page-all page to the 100 ms minimum.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A page-all read tops a short retry wait up to the minimum delay
func TestTiming_PageAllShortRetryTopsUpToMinimum(t *testing.T) {
	s := rebindSpec(t, pagingPageAllOp)
	tr := rebindReplies(
		pagingPage(pagingDocs(1, 2), nil),
		timingRetryAfter("0"),
		pagingPage(pagingDocs(3, 3), nil),
	)
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	if _, err := pagingRead(eng, "example-wiki.docs.all", nil); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[100ms 100ms]" {
		t.Fatalf("the waits are %s, want the page pacing and the retry, both 100 ms", got)
	}
}

// TestTiming_PageAllLongRetryKeptAsAsked binds a 429 wait longer than the minimum on a page-all page to the Retry-After
// time, with nothing added.
// @scenario specs/libs/retriever-engine/behaviours/timing-and-retries.feature: A page-all read keeps a retry wait longer than the minimum delay
func TestTiming_PageAllLongRetryKeptAsAsked(t *testing.T) {
	s := rebindSpec(t, pagingPageAllOp)
	tr := rebindReplies(
		pagingPage(pagingDocs(1, 2), nil),
		timingRetryAfter("2"),
		pagingPage(pagingDocs(3, 3), nil),
	)
	clock := &rebindClock{}
	eng := rebindEngine(t, s, tr, clock)
	if _, err := pagingRead(eng, "example-wiki.docs.all", nil); err != nil {
		t.Fatalf("the read fails: %v", err)
	}
	if got := fmt.Sprint(clock.recordedWaits()); got != "[100ms 2s]" {
		t.Fatalf("the waits are %s, want the page pacing and the Retry-After time alone", got)
	}
}
