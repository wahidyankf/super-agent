package mcp

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestAToolCallTimesItsCallOnTheInjectedClock checks the clock seam: each tool call reads the clock in Options, and its log
// line reports the duration on that clock. The clock here moves one second on each read, and a call reads it twice, at
// its start and at its end, so the line reports 1000 milliseconds and no test reads the real clock.
func TestAToolCallTimesItsCallOnTheInjectedClock(t *testing.T) {
	var logs bytes.Buffer
	reads := 0
	now := func() time.Time {
		reads++
		return time.Unix(int64(reads), 0)
	}
	gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-clock-1", Title: "Clock", URL: "/doc/clock", Markdown: "Body"}}
	server, err := NewServer(serviceWith(t, gw, mustSource(t, "outline")), Options{
		Version: "test",
		Logger:  slog.New(slog.NewTextHandler(&logs, nil)),
		Now:     now,
	})
	if err != nil {
		t.Fatal(err)
	}
	cs := clientOf(t, server)
	if _, isErr := call(t, cs, "get_document", map[string]any{"document_id": "doc-clock-1"}); isErr {
		t.Fatal("get_document failed")
	}
	if reads == 0 {
		t.Fatal("the tool call did not read the injected clock")
	}
	line := strings.TrimSpace(logs.String())
	if !strings.Contains(line, "duration_ms=1000") {
		t.Fatalf("log line %q does not report the duration on the injected clock", line)
	}
}
