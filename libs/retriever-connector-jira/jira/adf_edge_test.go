package jira

import (
	"encoding/json"
	"testing"
)

// TestRenderADFEdgeShapes pins the shapes that the main tables do not reach: a list item that is not a list item, a mention
// without its at sign, an emoji with no name, a date that is not a whole number, a code span across a line break, a code
// span that starts with a backtick, and a table row with no cells.
func TestRenderADFEdgeShapes(t *testing.T) {
	tests := []struct {
		name string
		in   json.RawMessage
		want string
	}{
		{
			name: "a list item that is a paragraph renders as its own line",
			in:   adfDoc(`{"type":"bulletList","content":[` + para("loose") + `]}`),
			want: "- loose",
		},
		{
			name: "a mention without its at sign gets one",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"mention","attrs":{"text":"Ann"}}]}`),
			want: "@Ann",
		},
		{
			name: "an emoji with neither text nor short name shows the unsupported marker",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"emoji","attrs":{}}]}`),
			want: "[unsupported content]",
		},
		{
			name: "a date that is not a whole number shows the unsupported marker",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"date","attrs":{"timestamp":1.5}}]}`),
			want: "[unsupported content]",
		},
		{
			name: "a code span across a line break keeps one line",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"a\nb","marks":[{"type":"code"}]}]}`),
			want: "`a b`",
		},
		{
			name: "a code span that starts with a backtick is padded",
			in:   adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"` + "`" + `x","marks":[{"type":"code"}]}]}`),
			want: "`` `x ``",
		},
		{
			name: "a table row with no cells is skipped",
			in: adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[]},` +
				`{"type":"tableRow","content":[{"type":"tableCell","content":[` + para("a") + `]}]}]}`),
			want: "| a |",
		},
		{
			name: "a link with an unsafe target shows only its text",
			in: adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[` +
				`{"type":"link","attrs":{"href":"https://e.x/a b"}}]}]}`),
			want: "x",
		},
		{
			name: "a second link on one text node is ignored",
			in: adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[` +
				`{"type":"link","attrs":{"href":"https://a.example/"}},` +
				`{"type":"link","attrs":{"href":"https://b.example/"}}]}]}`),
			want: "[x](<https://a.example/>)",
		},
	}
	runRenderTable(t, tests)
}
