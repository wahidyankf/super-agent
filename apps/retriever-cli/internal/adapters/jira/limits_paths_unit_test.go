package jira

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestACommentPageLongerThanItsLimitIsCutToIt proves a comments page that holds more comments than the limit answers only
// the limit, in the order Jira returned them.
func TestACommentPageLongerThanItsLimitIsCutToIt(t *testing.T) {
	body := `{"startAt":0,"maxResults":50,"total":2,"comments":[` +
		`{"id":"1","author":{"displayName":"A"},"created":"2026-11-03T01:30:00.000+0000","updated":"2026-11-03T01:30:00.000+0000","body":"one"},` +
		`{"id":"2","author":{"displayName":"B"},"created":"2026-11-03T01:31:00.000+0000","updated":"2026-11-03T01:31:00.000+0000","body":"two"}]}`
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, body), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	rec, err := readOp(gw, domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-12", "limit": 1})
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	items := rec.Items()
	if len(items) != 1 || items[0].Str("id") != "1" {
		t.Fatalf("items %d, want only the first comment", len(items))
	}
}

// TestACommentBodyThatIsNotADocumentIsUnexpected proves a comment whose body is neither a string nor an object is an
// unexpected reply, because its Markdown cannot be rendered.
func TestACommentBodyThatIsNotADocumentIsUnexpected(t *testing.T) {
	body := `{"startAt":0,"maxResults":50,"total":1,"comments":[` +
		`{"id":"1","author":{"displayName":"A"},"created":"2026-11-03T01:30:00.000+0000","updated":"2026-11-03T01:30:00.000+0000","body":5}]}`
	p := &probe{script: func(int, *http.Request) (*http.Response, error) {
		return reply(http.StatusOK, body), nil
	}}
	gw := probeGateway(t, p, newFakeClock(), newSecrets())
	if _, err := readOp(gw, domain.OpJiraCommentsList, map[string]any{"issueKey": "EXAMPLE-12"}); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestANestedBlockOverTheOutputBudgetMarksTheRenderTruncated proves a block that runs out of room while it is captured, here
// a block quote holding more text than MaxTextBytes, marks the whole render truncated and ends it with the marker.
func TestANestedBlockOverTheOutputBudgetMarksTheRenderTruncated(t *testing.T) {
	text := strings.Repeat("x", MaxTextBytes+64)
	raw := `{"type":"doc","content":[{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"` +
		text + `"}]}]}]}`
	out, err := RenderADF(json.RawMessage(raw))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if !strings.HasSuffix(out, truncatedMarker) {
		t.Fatalf("render does not end with %q: the nested block was not marked truncated", truncatedMarker)
	}
}
