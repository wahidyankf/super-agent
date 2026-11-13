package jira

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// adfDoc wraps block JSON in a document node, the shape Jira sends for a description.
func adfDoc(blocks string) json.RawMessage {
	return json.RawMessage(`{"type":"doc","version":1,"content":[` + blocks + `]}`)
}

// para returns a paragraph node with one plain text node.
func para(text string) string {
	return `{"type":"paragraph","content":[{"type":"text","text":` + jsonString(text) + `}]}`
}

// jsonString encodes s as a JSON string literal for a test fixture.
func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// TestRenderADFBlockNodes checks each block node type of the allow-list, one row per node.
func TestRenderADFBlockNodes(t *testing.T) {
	tests := []struct {
		name string
		in   json.RawMessage
		want string
	}{
		{
			name: "paragraph with plain text",
			in:   adfDoc(para("Hello world")),
			want: "Hello world",
		},
		{
			name: "two paragraphs are separated by a blank line",
			in:   adfDoc(para("one") + "," + para("two")),
			want: "one\n\ntwo",
		},
		{
			name: "heading level is shifted by two",
			in:   adfDoc(`{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Title"}]}`),
			want: "### Title",
		},
		{
			name: "heading level is capped at six",
			in:   adfDoc(`{"type":"heading","attrs":{"level":5},"content":[{"type":"text","text":"Deep"}]}`),
			want: "###### Deep",
		},
		{
			name: "heading without a level is level three",
			in:   adfDoc(`{"type":"heading","content":[{"type":"text","text":"Plain"}]}`),
			want: "### Plain",
		},
		{
			name: "bullet list",
			in: adfDoc(`{"type":"bulletList","content":[` +
				`{"type":"listItem","content":[` + para("one") + `]},` +
				`{"type":"listItem","content":[` + para("two") + `]}]}`),
			want: "- one\n- two",
		},
		{
			name: "ordered list starts at its order attribute",
			in: adfDoc(`{"type":"orderedList","attrs":{"order":3},"content":[` +
				`{"type":"listItem","content":[` + para("a") + `]},` +
				`{"type":"listItem","content":[` + para("b") + `]}]}`),
			want: "3. a\n4. b",
		},
		{
			name: "nested list is indented under its item",
			in: adfDoc(`{"type":"bulletList","content":[{"type":"listItem","content":[` +
				para("outer") + `,{"type":"bulletList","content":[{"type":"listItem","content":[` + para("inner") +
				`]}]}]}]}`),
			want: "- outer\n\n  - inner",
		},
		{
			name: "blockquote prefixes every line",
			in:   adfDoc(`{"type":"blockquote","content":[` + para("quoted") + `]}`),
			want: "> quoted",
		},
		{
			name: "code block with a language",
			in:   adfDoc(`{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"x := 1"}]}`),
			want: "```go\nx := 1\n```",
		},
		{
			name: "code block without a language",
			in:   adfDoc(`{"type":"codeBlock","content":[{"type":"text","text":"x"}]}`),
			want: "```\nx\n```",
		},
		{
			name: "code block with an unsafe language is written without it",
			in:   adfDoc(`{"type":"codeBlock","attrs":{"language":"go lang!"},"content":[{"type":"text","text":"x"}]}`),
			want: "```\nx\n```",
		},
		{
			name: "code block fence is longer than any backtick run inside",
			in:   adfDoc(`{"type":"codeBlock","content":[{"type":"text","text":"` + "```" + `"}]}`),
			want: "````\n```\n````",
		},
		{
			name: "rule",
			in:   adfDoc(`{"type":"rule"}`),
			want: "---",
		},
		{
			name: "panel shows its content",
			in:   adfDoc(`{"type":"panel","attrs":{"panelType":"info"},"content":[` + para("note") + `]}`),
			want: "note",
		},
		{
			name: "expand shows its title in bold and then its content",
			in:   adfDoc(`{"type":"expand","attrs":{"title":"More"},"content":[` + para("body") + `]}`),
			want: "**More**\n\nbody",
		},
		{
			name: "nested expand without a title shows the content",
			in:   adfDoc(`{"type":"nestedExpand","content":[` + para("inside") + `]}`),
			want: "inside",
		},
		{
			name: "task list shows checked and open items",
			in: adfDoc(`{"type":"taskList","content":[` +
				`{"type":"taskItem","attrs":{"state":"DONE"},"content":[{"type":"text","text":"done"}]},` +
				`{"type":"taskItem","attrs":{"state":"TODO"},"content":[{"type":"text","text":"todo"}]}]}`),
			want: "- [x] done\n- [ ] todo",
		},
		{
			name: "decision list shows each decision",
			in: adfDoc(`{"type":"decisionList","content":[` +
				`{"type":"decisionItem","content":[{"type":"text","text":"agreed"}]}]}`),
			want: "- agreed",
		},
		{
			name: "table with a header row and one body row",
			in: adfDoc(`{"type":"table","content":[` +
				`{"type":"tableRow","content":[` +
				`{"type":"tableHeader","content":[` + para("A") + `]},` +
				`{"type":"tableHeader","content":[` + para("B") + `]}]},` +
				`{"type":"tableRow","content":[` +
				`{"type":"tableCell","content":[` + para("1") + `]},` +
				`{"type":"tableCell","content":[` + para("2") + `]}]}]}`),
			want: "| A | B |\n| --- | --- |\n| 1 | 2 |",
		},
		{
			name: "table cell line breaks become spaces and pipes are escaped",
			in: adfDoc(`{"type":"table","content":[` +
				`{"type":"tableRow","content":[{"type":"tableCell","content":[` + para("a|b") + `,` + para("c") +
				`]}]}]}`),
			want: "| a\\|b c |\n| --- |",
		},
		{
			name: "media shows its alt text as an attachment without a URL",
			in: adfDoc(`{"type":"mediaSingle","content":[` +
				`{"type":"media","attrs":{"type":"file","id":"secret-media-id","alt":"a.png"}}]}`),
			want: "[attachment: a.png]",
		},
		{
			name: "media without alt text is named file",
			in: adfDoc(`{"type":"mediaGroup","content":[` +
				`{"type":"media","attrs":{"type":"file","id":"secret-media-id"}}]}`),
			want: "[attachment: file]",
		},
		{
			name: "an unknown block with text shows the text",
			in:   adfDoc(`{"type":"futureBlock","content":[` + para("kept") + `]}`),
			want: "kept",
		},
		{
			name: "an unknown block with no content shows the unsupported marker",
			in:   adfDoc(`{"type":"futureBlock"}`),
			want: "[unsupported content]",
		},
		{
			name: "a text node directly under the document is a paragraph",
			in:   adfDoc(`{"type":"text","text":"bare"}`),
			want: "bare",
		},
	}
	runRenderTable(t, tests)
}

// TestRenderADFInlineNodes checks each inline node and mark of the allow-list.
func TestRenderADFInlineNodes(t *testing.T) {
	tests := []struct {
		name string
		in   json.RawMessage
		want string
	}{
		{
			name: "strong mark",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"bold","marks":[{"type":"strong"}]}]}`),
			want: "**bold**",
		},
		{
			name: "em mark",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"it","marks":[{"type":"em"}]}]}`),
			want: "*it*",
		},
		{
			name: "strike mark",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"gone","marks":[{"type":"strike"}]}]}`),
			want: "~~gone~~",
		},
		{
			name: "code mark",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"x*y","marks":[{"type":"code"}]}]}`),
			want: "`x*y`",
		},
		{
			name: "code mark with a backtick inside uses a longer fence",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"a` + "`" + `b","marks":[{"type":"code"}]}]}`),
			want: "``a`b``",
		},
		{
			name: "https link mark",
			in: adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"site","marks":[` +
				`{"type":"link","attrs":{"href":"https://example.com/a"}}]}]}`),
			want: "[site](<https://example.com/a>)",
		},
		{
			name: "mailto link mark",
			in: adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"mail","marks":[` +
				`{"type":"link","attrs":{"href":"mailto:ann@example.com"}}]}]}`),
			want: "[mail](<mailto:ann@example.com>)",
		},
		{
			name: "javascript link mark shows only the escaped text",
			in: adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"click","marks":[` +
				`{"type":"link","attrs":{"href":"javascript:alert(1)"}}]}]}`),
			want: "click",
		},
		{
			name: "an unknown mark is ignored",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"red","marks":[{"type":"textColor"}]}]}`),
			want: "red",
		},
		{
			name: "hard break",
			in: adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"a"},{"type":"hardBreak"},` +
				`{"type":"text","text":"b"}]}`),
			want: "a\\\nb",
		},
		{
			name: "mention shows its text and never the account ID",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"mention","attrs":{"id":"acc-secret-123","text":"@Ann"}}]}`),
			want: "@Ann",
		},
		{
			name: "mention without text shows a placeholder",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"mention","attrs":{"id":"acc-secret-123"}}]}`),
			want: "@unknown",
		},
		{
			name: "emoji shows its text",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"emoji","attrs":{"shortName":":smile:","text":"☺"}}]}`),
			want: "☺",
		},
		{
			name: "emoji without text shows its short name",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"emoji","attrs":{"shortName":":smile:"}}]}`),
			want: ":smile:",
		},
		{
			name: "status shows its text in brackets",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"status","attrs":{"text":"DONE","color":"green"}}]}`),
			want: "[DONE]",
		},
		{
			name: "date shows the UTC day of its epoch milliseconds",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"date","attrs":{"timestamp":"1767225600000"}}]}`),
			want: "2026-01-01",
		},
		{
			name: "date with a number shows the same day",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"date","attrs":{"timestamp":1767225600000}}]}`),
			want: "2026-01-01",
		},
		{
			name: "date that does not parse shows the unsupported marker",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"date","attrs":{"timestamp":"soon"}}]}`),
			want: "[unsupported content]",
		},
		{
			name: "inline card with an https URL shows the URL as an autolink",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"inlineCard","attrs":{"url":"https://example.com/x"}}]}`),
			want: "<https://example.com/x>",
		},
		{
			name: "card with a javascript URL is removed",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"inlineCard","attrs":{"url":"javascript:alert(1)"}}]}`),
			want: "[link removed]",
		},
		{
			name: "block card is a paragraph",
			in:   adfDoc(`{"type":"blockCard","attrs":{"url":"https://example.com/b"}},` + para("after")),
			want: "<https://example.com/b>\n\nafter",
		},
		{
			name: "embed card is a block link",
			in:   adfDoc(`{"type":"embedCard","attrs":{"url":"https://example.com/e"}}`),
			want: "<https://example.com/e>",
		},
		{
			name: "inline media is an attachment",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"mediaInline","attrs":{"id":"m-1","alt":"chart"}}]}`),
			want: "[attachment: chart]",
		},
		{
			name: "unknown inline with text children shows the text",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"futureInline","content":[{"type":"text","text":"deep"}]}]}`),
			want: "deep",
		},
		{
			name: "unknown inline with no content shows the unsupported marker",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"futureInline"}]}`),
			want: "[unsupported content]",
		},
	}
	runRenderTable(t, tests)
}

// TestRenderADFHostileText checks that every text node is escaped and cleaned, one row per hostile string.
func TestRenderADFHostileText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "HTML tags are escaped", text: "<script>alert(1)</script>", want: "\\<script\\>alert(1)\\</script\\>"},
		{name: "a heading marker is escaped", text: "# not a heading", want: "\\# not a heading"},
		{name: "a bullet at line start is escaped", text: "- item", want: "\\- item"},
		{name: "a plus at line start is escaped", text: "+ item", want: "\\+ item"},
		{name: "an equals sign at line start is escaped", text: "=== title", want: "\\=== title"},
		{name: "a numbered list marker is escaped", text: "1. step", want: "1\\. step"},
		{name: "a parenthesised number marker is escaped", text: "2) step", want: "2\\) step"},
		{name: "a number not followed by a marker is kept", text: "2024 was good", want: "2024 was good"},
		{name: "a link is not formed", text: "[x](http://evil)", want: "\\[x\\](http://evil)"},
		{name: "an image is not formed", text: "![i](http://evil)", want: "\\!\\[i\\](http://evil)"},
		{name: "a table cell pipe is escaped", text: "a | b", want: "a \\| b"},
		{name: "an entity is not decoded", text: "&amp; &lt;", want: "\\&amp; \\&lt;"},
		{name: "a backtick is escaped", text: "`code`", want: "\\`code\\`"},
		{name: "an escape sequence loses its control byte", text: "\x1b[31mred\x1b[0m", want: "\\[31mred\\[0m"},
		{name: "a bidirectional override is removed", text: "abc\u202Edef", want: "abcdef"},
		{name: "a zero-width space is removed", text: "a\u200Bb", want: "ab"},
		{name: "a byte order mark is removed", text: "\uFEFFtext", want: "text"},
		{name: "a NUL byte is removed", text: "a\x00b", want: "ab"},
		{name: "a carriage return is removed", text: "a\rb", want: "ab"},
		{name: "a tab is kept", text: "a\tb", want: "a\tb"},
		{name: "a line break in text keeps the next line plain", text: "one\ntwo", want: "one\ntwo"},
		{name: "leading spaces at line start are dropped", text: "   indented", want: "indented"},
		{name: "a mention-like text is plain", text: "@Ann", want: "@Ann"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := adfDoc(para(tt.text))
			got, err := RenderADF(in)
			if err != nil {
				t.Fatalf("RenderADF: %v", err)
			}
			if got != tt.want {
				t.Fatalf("RenderADF(%q) = %q, want %q", tt.text, got, tt.want)
			}
			assertNoControlOrBidi(t, got)
		})
	}
}

// TestRenderADFWholeInputShapes checks the inputs that are not a document node.
func TestRenderADFWholeInputShapes(t *testing.T) {
	tests := []struct {
		name string
		in   json.RawMessage
		want string
	}{
		{name: "a missing value is empty", in: nil, want: ""},
		{name: "a null value is empty", in: json.RawMessage("null"), want: ""},
		{name: "a blank value is empty", in: json.RawMessage("  \n"), want: ""},
		{name: "a plain JSON string is one escaped paragraph", in: json.RawMessage(`"1. not a list"`), want: "1\\. not a list"},
		{name: "an empty document is empty", in: json.RawMessage(`{"type":"doc","content":[]}`), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderADF(tt.in)
			if err != nil {
				t.Fatalf("RenderADF: %v", err)
			}
			if got != tt.want {
				t.Fatalf("RenderADF = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderADFRefusesMalformedJSON checks that input that is not JSON, or not a JSON node, is an unexpected response.
func TestRenderADFRefusesMalformedJSON(t *testing.T) {
	for _, bad := range []string{`{"type":`, `[1,2]`, `{"type":"paragraph","content":"text"}`, `true`} {
		_, err := RenderADF(json.RawMessage(bad))
		if !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Errorf("RenderADF(%s) = %v, want ErrUnexpectedResponse", bad, err)
		}
	}
}

// TestRenderADFDepthLimitTruncatesAndSaysSo checks that a tree deeper than the depth limit renders what fits and ends with the
// truncation marker, and that the text below the limit is not shown.
func TestRenderADFDepthLimitTruncatesAndSaysSo(t *testing.T) {
	shallow := nestedPanels(maxADFDepth-3, "shallow-text")
	got, err := RenderADF(shallow)
	if err != nil {
		t.Fatalf("RenderADF(shallow): %v", err)
	}
	if got != "shallow-text" {
		t.Fatalf("shallow tree = %q, want its text", got)
	}

	deep := nestedPanels(maxADFDepth+20, "deep-text")
	got, err = RenderADF(deep)
	if err != nil {
		t.Fatalf("RenderADF(deep): %v", err)
	}
	if strings.Contains(got, "deep-text") {
		t.Fatalf("text below the depth limit is shown: %q", got)
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Fatalf("deep tree has no truncation marker: %q", got)
	}
}

// TestRenderADFStopsAtTheNodeLimit checks that more nodes than the limit render what fits and say so.
func TestRenderADFStopsAtTheNodeLimit(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"type":"doc","content":[`)
	for i := 0; i < maxADFNodes+10; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"type":"rule"}`)
	}
	b.WriteString(`]}`)
	got, err := RenderADF(json.RawMessage(b.String()))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Fatalf("a tree over the node limit has no truncation marker (tail %q)", tail(got))
	}
	if len(got) > MaxTextBytes {
		t.Fatalf("output %d bytes is over the cap", len(got))
	}
}

// TestRenderADFCapsTheOutputAtOneMiB checks that a text longer than the cap is cut at a whole character, below the cap,
// with the marker.
func TestRenderADFCapsTheOutputAtOneMiB(t *testing.T) {
	long := strings.Repeat("é", MaxTextBytes) // two bytes per character, so the cut lands on a boundary
	got, err := RenderADF(adfDoc(para(long)))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if len(got) > MaxTextBytes {
		t.Fatalf("output is %d bytes, want at most %d", len(got), MaxTextBytes)
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Fatalf("capped output has no marker (tail %q)", tail(got))
	}
	if !validUTF8(got) {
		t.Fatalf("capped output is not valid UTF-8")
	}
}

// TestRenderADFDropsAnOversizedCodeBlockWhole checks that a code block that does not fit is not written half-open.
func TestRenderADFDropsAnOversizedCodeBlockWhole(t *testing.T) {
	code := strings.Repeat("x", MaxTextBytes)
	in := adfDoc(`{"type":"codeBlock","content":[{"type":"text","text":` + jsonString(code) + `}]}`)
	got, err := RenderADF(in)
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if strings.Contains(got, "```") {
		t.Fatalf("an oversized code block left a fence open or partial (tail %q)", tail(got))
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Fatalf("missing marker (tail %q)", tail(got))
	}
}

// nestedPanels returns a document of depth nested panels with leaf as the text of the innermost paragraph.
func nestedPanels(depth int, leaf string) json.RawMessage {
	inner := para(leaf)
	for i := 0; i < depth; i++ {
		inner = `{"type":"panel","content":[` + inner + `]}`
	}
	return adfDoc(inner)
}

// runRenderTable runs a table of RenderADF cases and checks each output for control and bidirectional characters.
func runRenderTable(t *testing.T, tests []struct {
	name string
	in   json.RawMessage
	want string
}) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderADF(tt.in)
			if err != nil {
				t.Fatalf("RenderADF: %v", err)
			}
			if got != tt.want {
				t.Fatalf("RenderADF = %q, want %q", got, tt.want)
			}
			assertNoControlOrBidi(t, got)
		})
	}
}

// assertNoControlOrBidi fails when out holds a control, a zero-width, or a bidirectional character, other than newline and tab.
func assertNoControlOrBidi(t *testing.T, out string) {
	t.Helper()
	if stripControls(out) != out {
		t.Fatalf("output holds a control or bidirectional character: %q", out)
	}
}

// tail returns the last 40 bytes of s, for a failure message.
func tail(s string) string {
	if len(s) > 40 {
		return s[len(s)-40:]
	}
	return s
}

// validUTF8 reports whether s is valid UTF-8.
func validUTF8(s string) bool {
	return utf8.ValidString(s)
}
