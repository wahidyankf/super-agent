package engine

import (
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// editorDepthCap is the deepest editor node whose text is read, mirroring the Go comment projection. A node nested
// deeper is skipped, so a hostile reply cannot drive the walk without bound. A real comment is a few levels deep.
const editorDepthCap = 64

// editorText returns the plain text of editor data: the ProseMirror JSON that a wiki keeps for a comment, read as the
// Go comment projection reads it. The words of each text block end with a line break, a hard break is a line break,
// and a mention reads as its label with an at sign. Marks, other attributes, and the node types are dropped, so the
// result is only the text a reader would see. A plain string passes through unchanged, so a field that falls back to
// raw text keeps it. A missing or null value maps to null, like any other missing optional field. Any other scalar
// is an unexpected response, so a whole value is never copied. The text is charged to the shared output budget, like
// any mapped text.
func editorText(raw any, budget *int) (any, error) {
	if raw == nil {
		return nil, nil
	}
	if s, ok := raw.(string); ok {
		return cleanText(s, budget)
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, domain.ErrUnexpectedResponse
	}
	var b strings.Builder
	writeEditorNode(&b, m, 0)
	out := strings.TrimRight(b.String(), "\n")
	if len(out) > *budget {
		return nil, domain.ErrReplyTooLarge
	}
	*budget -= len(out)
	return out, nil
}

// writeEditorNode writes the text of one editor node and of its children, in order. depth is the nesting of the
// node. A value that is not an editor node writes nothing, and a node nested deeper than the cap is skipped.
func writeEditorNode(b *strings.Builder, raw any, depth int) {
	if depth > editorDepthCap {
		return
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return
	}
	typ, _ := m["type"].(string)
	switch typ {
	case "hard_break":
		b.WriteString("\n")
		return
	case "mention":
		// A mention is inline: its label is its text, and it never ends a line.
		if label := editorMentionLabel(m); label != "" {
			b.WriteString("@" + label)
		}
		return
	}
	if text, ok := m["text"].(string); ok {
		b.WriteString(text)
	}
	children, _ := m["content"].([]any)
	for _, child := range children {
		writeEditorNode(b, child, depth+1)
	}
	if editorEndsBlock(m, children) {
		b.WriteString("\n")
	}
}

// editorMentionLabel returns the label of a mention node, or "" when it has none or the attributes are not an object.
func editorMentionLabel(m map[string]any) string {
	attrs, ok := m["attrs"].(map[string]any)
	if !ok {
		return ""
	}
	label, _ := attrs["label"].(string)
	return label
}

// editorInlineTypes are the node types that sit within a line, so a node holding only them is a text block.
var editorInlineTypes = map[string]bool{"text": true, "hard_break": true, "mention": true}

// editorEndsBlock reports whether a node is a text block, whose text ends with a line break. A block is a node whose
// children are all inline, or that holds no children at all, such as an empty paragraph. The document and the inline
// nodes are never blocks.
func editorEndsBlock(m map[string]any, children []any) bool {
	typ, _ := m["type"].(string)
	if typ == "" || typ == "doc" || editorInlineTypes[typ] {
		return false
	}
	if len(children) == 0 {
		return true
	}
	for _, child := range children {
		c, ok := child.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := c["type"].(string); !editorInlineTypes[t] {
			return false
		}
	}
	return true
}
