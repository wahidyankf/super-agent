package engine_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine/replay"
)

// TestPacing_ASingleReadRetryWaitsTheMinDelay retries a rate-limited single read after the pacing delay, then
// succeeds. A 429 wait shorter than the delay is topped up to it, so a single read such as the Jira whoami keeps the
// spacing of the Go gateway it replaces: three requests leave two waits of 100 ms.
func TestPacing_ASingleReadRetryWaitsTheMinDelay(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/search",
		replay.Reply{Status: http.StatusTooManyRequests, Header: map[string][]string{"Retry-After": {"0"}}},
		replay.Reply{Status: http.StatusTooManyRequests, Header: map[string][]string{"Retry-After": {"0"}}},
		page(`{"id":"doc-1"}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.search", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := len(stub.Requests()); got != 3 {
		t.Fatalf("got %d requests, want 3", got)
	}
	want := []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}
	if got := clock.waits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}

// TestPacing_ASingleReadKeepsALongerRetryWait checks that a 429 wait longer than the pacing delay is kept as the
// reply asks: a longer Retry-After already covers the delay, so nothing is added.
func TestPacing_ASingleReadKeepsALongerRetryWait(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/search",
		replay.Reply{Status: http.StatusTooManyRequests, Header: map[string][]string{"Retry-After": {"1"}}},
		page(`{"id":"doc-1"}`))
	if _, err := eng.Read(context.Background(), "example-wiki.docs.search", domain.NewArgs(nil)); err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []time.Duration{time.Second}
	if got := clock.waits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}

// TestPacing_OffsetPagesAreSpacedToTheMinDelay reads three offset pages, the last one short. The engine waits the
// pacing delay before every page after the first, so the three requests leave two waits of 100 ms.
func TestPacing_OffsetPagesAreSpacedToTheMinDelay(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=0", page(`{"items":[{"id":"doc-1"},{"id":"doc-2"}]}`))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=2", page(`{"items":[{"id":"doc-3"},{"id":"doc-4"}]}`))
	stub.Script("GET", "/v1/docs/page?limit=2&offset=4", page(`{"items":[{"id":"doc-5"}]}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.page", domain.NewArgs(map[string]any{"limit": 2}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 5 {
		t.Fatalf("got %d items, want 5", len(rec.Items()))
	}
	want := []time.Duration{100 * time.Millisecond, 100 * time.Millisecond}
	if got := clock.waits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}

// TestPacing_CursorPagesAreSpacedToTheMinDelay reads two cursor pages. The engine waits the pacing delay before the
// second page, so the two requests leave one wait of 100 ms.
func TestPacing_CursorPagesAreSpacedToTheMinDelay(t *testing.T) {
	eng, stub, clock := newStubEngine(t, loadSpec(t))
	stub.Script("GET", "/v1/docs?limit=20", page(`{"items":[{"id":"doc-1"}],"next_cursor":"c2"}`))
	stub.Script("GET", "/v1/docs?cursor=c2&limit=20", page(`{"items":[{"id":"doc-2"}],"next_cursor":null}`))
	rec, err := eng.Read(context.Background(), "example-wiki.docs.list", domain.NewArgs(nil))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(rec.Items()) != 2 {
		t.Fatalf("got %d items, want 2", len(rec.Items()))
	}
	want := []time.Duration{100 * time.Millisecond}
	if got := clock.waits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waits = %v, want %v", got, want)
	}
}
