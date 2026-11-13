package outline

import "strings"

// maxEditorDepth is the deepest editor node whose text is read. A node nested deeper is skipped, so a hostile reply
// cannot drive the walk without bound. A real comment is a few levels deep.
const maxEditorDepth = 64

// plainText returns the words of one editor node (the ProseMirror JSON that Outline keeps for a comment), with a line
// break after each text block and each hard break. A mention reads as its label with an @ sign. Marks, other attributes,
// and the node types are dropped, so the result is only the text a reader would see. A value that is not an editor node
// is "", and a node nested deeper than maxEditorDepth is skipped.
func plainText(node any) string {
	var b strings.Builder
	writeNodeText(&b, node, 0)
	return strings.TrimRight(b.String(), "\n")
}

// writeNodeText writes the text of one node and of its children, in order. depth is the nesting of the node.
func writeNodeText(b *strings.Builder, raw any, depth int) {
	if depth > maxEditorDepth {
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
		if label := mentionLabel(m); label != "" {
			b.WriteString("@" + label)
		}
		return
	}
	if text, ok := m["text"].(string); ok {
		b.WriteString(text)
	}
	children, _ := m["content"].([]any)
	for _, child := range children {
		writeNodeText(b, child, depth+1)
	}
	if endsBlock(m, children) {
		b.WriteString("\n")
	}
}

// mentionLabel returns the label of a mention node, or "" when it has none or the attributes are not an object.
func mentionLabel(m map[string]any) string {
	attrs, ok := m["attrs"].(map[string]any)
	if !ok {
		return ""
	}
	label, _ := attrs["label"].(string)
	return label
}

// inlineTypes are the node types that sit within a line, so a node holding only them is a text block.
var inlineTypes = map[string]bool{"text": true, "hard_break": true, "mention": true}

// endsBlock reports whether a node is a text block, which the text of a line ends with a break. A block is a node whose
// children are all inline, or that holds no children at all, such as an empty paragraph. The document and the inline
// nodes are never blocks.
func endsBlock(m map[string]any, children []any) bool {
	typ, _ := m["type"].(string)
	if typ == "" || typ == "doc" || inlineTypes[typ] {
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
		if t, _ := c["type"].(string); !inlineTypes[t] {
			return false
		}
	}
	return true
}
