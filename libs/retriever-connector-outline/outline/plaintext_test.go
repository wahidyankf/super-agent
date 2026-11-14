package outline

import (
	"strings"
	"testing"
)

// nestedEditor returns an editor document whose only text sits at the bottom of depth nested block nodes.
func nestedEditor(depth int, text string) string {
	var b strings.Builder
	for i := 0; i < depth; i++ {
		b.WriteString(`{"type":"blockquote","content":[`)
	}
	b.WriteString(`{"type":"text","text":"` + text + `"}`)
	for i := 0; i < depth; i++ {
		b.WriteString(`]}`)
	}
	return b.String()
}

// TestPlainTextKeepsAMentionAsItsLabelAndGluesNoParagraphs checks that a mention is read as @label, so a paragraph that
// holds one still ends its line, and the next paragraph does not run on from it.
func TestPlainTextKeepsAMentionAsItsLabelAndGluesNoParagraphs(t *testing.T) {
	node := `{"type":"doc","content":[` +
		`{"type":"paragraph","content":[{"type":"text","text":"Hi "},` +
		`{"type":"mention","attrs":{"id":"u-1","type":"user","label":"bob"}}]},` +
		`{"type":"paragraph","content":[{"type":"text","text":"Next"}]}]}`
	if got := plainText(decodeJSON(t, node)); got != "Hi @bob\nNext" {
		t.Fatalf("plainText = %q, want %q", got, "Hi @bob\nNext")
	}
	// A mention with no label has no text of its own, and it does not glue the lines either.
	bare := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"id":"u-2"}}]},` +
		`{"type":"paragraph","content":[{"type":"text","text":"After"}]}]}`
	if got := plainText(decodeJSON(t, bare)); got != "\nAfter" {
		t.Fatalf("plainText(bare mention) = %q, want %q", got, "\nAfter")
	}
}

// TestPlainTextStopsAtDepth64 checks the depth limit: text nested deeper than 64 nodes is not read, so a hostile reply
// cannot drive the walk without bound. Text within the limit is kept.
func TestPlainTextStopsAtDepth64(t *testing.T) {
	shallow := decodeJSON(t, nestedEditor(10, "shallow"))
	if got := plainText(shallow); got != "shallow" {
		t.Fatalf("plainText(depth 10) = %q, want the text kept", got)
	}
	// 2000 blocks are 4000 levels of JSON nesting, which encoding/json reads (it refuses beyond 10000).
	deep := decodeJSON(t, nestedEditor(2000, "buried"))
	if got := plainText(deep); strings.Contains(got, "buried") {
		t.Fatalf("plainText(depth 2000) read the text below the limit: %.40q", got)
	}
}

// TestPlainTextIgnoresWrongTypes checks that a node of the wrong type is read as no text: a number, a list where a
// node is expected, a content that is not a list, and a text that is not text. None of them panics.
func TestPlainTextIgnoresWrongTypes(t *testing.T) {
	cases := []string{
		`{"type":"paragraph","content":"oops"}`,
		`{"type":3,"content":[{"type":"text","text":5}]}`,
		`[1,2,3]`,
		`{"type":"doc","content":[null,7,"x",{"type":"text","text":{"nested":true}}]}`,
		`{"type":"mention","attrs":"nope"}`,
		`{"type":"mention","attrs":{"label":7}}`,
		`{}`,
	}
	for _, node := range cases {
		if got := plainText(decodeJSON(t, node)); got != "" {
			t.Fatalf("plainText(%s) = %q, want no text", node, got)
		}
	}
}

// TestAReplyNestedBeyondTheJSONLimitIsUnexpected checks the reply boundary: a body nested 10001 levels deep is refused
// by the JSON decoder, so the projection never sees it and the caller gets an unexpected reply.
func TestAReplyNestedBeyondTheJSONLimitIsUnexpected(t *testing.T) {
	body := strings.Repeat("[", 10001) + strings.Repeat("]", 10001)
	var out any
	if _, _, err := interpret(methodDocumentsInfo, 200, "", []byte(body), &out); err == nil {
		t.Fatal("interpret(10001 levels) error = nil, want an unexpected reply")
	}
}
