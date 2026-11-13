package mcp

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestCutPageBoundaries checks the cap with ASCII: exactly the cap is one whole page, one byte over is cut at the cap.
func TestCutPageBoundaries(t *testing.T) {
	exact := strings.Repeat("a", pageCap)
	if text, _, truncated := cutPage(exact, 0); truncated || text != exact {
		t.Fatalf("exact cap: truncated=%v, length %d", truncated, len(text))
	}
	over := strings.Repeat("a", pageCap+1)
	text, next, truncated := cutPage(over, 0)
	if !truncated || len(text) != pageCap || next != pageCap {
		t.Fatalf("one over: truncated=%v length=%d next=%d", truncated, len(text), next)
	}
}

// TestCutPageNeverSplitsACharacter checks three-byte characters: 32768 is not a multiple of three, so the cut moves back
// to a boundary, and the pages join to the original text.
func TestCutPageNeverSplitsACharacter(t *testing.T) {
	doc := strings.Repeat("€", 11000)
	var joined strings.Builder
	offset := 0
	for pages := 0; ; pages++ {
		if pages > 100 {
			t.Fatal("paging did not end")
		}
		text, next, truncated := cutPage(doc, offset)
		if !utf8.ValidString(text) || len(text) > pageCap {
			t.Fatalf("page %d is not valid UTF-8 within the cap (length %d)", pages, len(text))
		}
		joined.WriteString(text)
		if !truncated {
			break
		}
		offset = next
	}
	if joined.String() != doc {
		t.Fatal("the pages do not join to the original document")
	}
}

// TestCutPageMovesAnOffsetInsideACharacterForward checks that a start inside a character begins at the next one.
func TestCutPageMovesAnOffsetInsideACharacterForward(t *testing.T) {
	doc := "€uro" // the euro sign is three bytes, so offset 1 is inside it
	text, _, truncated := cutPage(doc, 1)
	if truncated || text != "uro" {
		t.Fatalf("offset 1 gave %q, truncated=%v", text, truncated)
	}
}

// TestCutPageAtTheEnd checks that an offset equal to the length gives an empty last page.
func TestCutPageAtTheEnd(t *testing.T) {
	text, _, truncated := cutPage("abc", 3)
	if text != "" || truncated {
		t.Fatalf("offset at the end gave %q, truncated=%v", text, truncated)
	}
}

// TestCutPageAlwaysMovesForward checks that a page of text with no character boundary inside the cap still ends after its
// start, so a client that follows next_offset never reads the same empty page again.
func TestCutPageAlwaysMovesForward(t *testing.T) {
	doc := "a" + strings.Repeat("\x80", pageCap+10) // one byte that starts a character, then continuation bytes only
	text, next, truncated := cutPage(doc, 0)
	if !truncated || next <= 0 || text == "" {
		t.Fatalf("page from 0 is %d bytes with next=%d and truncated=%v, want the page to move forward", len(text), next, truncated)
	}
}
