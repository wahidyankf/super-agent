package cli

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// hostileTitle is wiki text that tries to set the terminal title and to clear the screen. Its ESC and BEL bytes must
// never reach a text-mode answer.
const hostileTitle = "Plan\x1b]0;pwned\x07 \x1b[2J\x9b31m\x7fend"

// hasTerminalControl reports whether s holds a control that a terminal would act on: a C0 control other than tab and
// line feed, DEL, or a C1 control (U+0080 to U+009F).
func hasTerminalControl(s string) bool {
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
			return true
		}
	}
	return false
}

// TestWriteTextNeverPrintsATerminalControlFromWikiText checks every text-mode shape: an object, a list, a tree, and a
// body. Each one prints the hostile title or body with no control byte left in it.
func TestWriteTextNeverPrintsATerminalControlFromWikiText(t *testing.T) {
	collections, _ := operations.Lookup(domain.OpCollectionsList)
	info, _ := operations.Lookup(domain.OpCollectionsInfo)
	tree, _ := operations.Lookup(domain.OpCollectionsDocuments)
	body, _ := operations.Lookup(domain.OpDocumentsGet)

	cases := []struct {
		name string
		row  operations.Row
		rec  domain.Record
	}{
		{
			name: "an object prints each field with the controls removed",
			row:  info,
			rec:  domain.NewRecord(domain.Field{Name: "id", Value: "col-1"}, domain.Field{Name: "name", Value: hostileTitle}),
		},
		{
			name: "a list prints each row with the controls removed",
			row:  collections,
			rec: domain.NewList([]domain.Record{
				domain.NewRecord(domain.Field{Name: "id", Value: "col-1"}, domain.Field{Name: "name", Value: hostileTitle}),
			}),
		},
		{
			name: "a tree prints each node with the controls removed",
			row:  tree,
			rec: domain.NewList([]domain.Record{
				domain.NewRecord(
					domain.Field{Name: "id", Value: "doc-1"},
					domain.Field{Name: "title", Value: hostileTitle},
					domain.Field{Name: "url", Value: "/doc/plan"},
					domain.Field{Name: "parentId", Value: nil},
					domain.Field{Name: "depth", Value: 0},
				),
			}),
		},
		{
			name: "a body prints its Markdown with the controls removed",
			row:  body,
			rec: domain.NewRecord(
				domain.Field{Name: "id", Value: "doc-1"},
				domain.Field{Name: "title", Value: "Plan"},
				domain.Field{Name: "url", Value: "/doc/plan"},
				domain.Field{Name: "collectionId", Value: "col-1"},
				domain.Field{Name: "markdown", Value: "# Plan\n\tindented " + hostileTitle + "\n"},
			),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeText(&buf, tc.row, tc.rec)
			out := buf.String()
			if strings.Contains(out, "\x1b") {
				t.Fatalf("text output holds an ESC byte: %q", out)
			}
			if hasTerminalControl(out) {
				t.Fatalf("text output holds a terminal control: %q", out)
			}
			if !utf8.ValidString(out) {
				t.Fatalf("text output is not valid UTF-8: %q", out)
			}
			if !strings.Contains(out, "pwned") {
				t.Fatalf("the visible text of the title was lost: %q", out)
			}
		})
	}
}

// TestWriteTreeCapsTheIndentOfADeepNode checks that a node deeper than the indent cap is printed with the cap's indent,
// so a hostile depth never makes a line of unbounded width (finding a4).
func TestWriteTreeCapsTheIndentOfADeepNode(t *testing.T) {
	tree, _ := operations.Lookup(domain.OpCollectionsDocuments)
	deep := domain.NewList([]domain.Record{
		domain.NewRecord(
			domain.Field{Name: "id", Value: "doc-1"},
			domain.Field{Name: "title", Value: "Deep"},
			domain.Field{Name: "url", Value: "/doc/deep"},
			domain.Field{Name: "parentId", Value: nil},
			domain.Field{Name: "depth", Value: 100000},
		),
	})
	var buf bytes.Buffer
	writeText(&buf, tree, deep)
	line := strings.TrimRight(buf.String(), "\n")
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 2*maxIndentDepth {
		t.Fatalf("indent = %d spaces, want at most %d", indent, 2*maxIndentDepth)
	}
}

// TestWriteTextKeepsTabsAndNewlinesOfABody checks that a body keeps its line structure: a line feed and a tab are not
// controls for this purpose, so the Markdown still reads as Markdown.
func TestWriteTextKeepsTabsAndNewlinesOfABody(t *testing.T) {
	body, _ := operations.Lookup(domain.OpDocumentsGet)
	rec := domain.NewRecord(domain.Field{Name: "markdown", Value: "a\tb\nc\n"})
	var buf bytes.Buffer
	writeText(&buf, body, rec)
	if got := buf.String(); got != "a\tb\nc\n" {
		t.Fatalf("body = %q, want the tab and the lines kept", got)
	}
}

// TestJSONOutputKeepsTheWikiTextAsItIs checks the boundary: the JSON document is not sanitized, because a JSON string
// escapes its controls, so the raw value stays the value a client decodes.
func TestJSONOutputKeepsTheWikiTextAsItIs(t *testing.T) {
	info, _ := operations.Lookup(domain.OpCollectionsInfo)
	rec := domain.NewRecord(domain.Field{Name: "id", Value: "col-1"}, domain.Field{Name: "name", Value: hostileTitle})
	doc := jsonDocument(info, rec)
	if got := doc["name"]; got != hostileTitle {
		t.Fatalf("JSON name = %q, want the raw wiki text", got)
	}
}
