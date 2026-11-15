// This file turns an Atlassian Document Format (ADF) value into Markdown. ADF is a JSON tree that many people write, so it is
// hostile input: RenderADF is a pure function with no I/O. It renders known nodes by an explicit switch, escapes all text, keeps
// only web and mail links, never fetches media, and stops at fixed depth, node, and size limits. The engine serves it as the
// markdown-from-adf converter, a pure function of the reply. The Jira connector keeps its own copy until it retires.
package engine

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

const (
	// MaxTextBytes is the most bytes RenderADF returns, including the truncation marker.
	MaxTextBytes = 1 << 20
	// maxADFDepth is the deepest node that is rendered. The document node is at depth 1.
	maxADFDepth = 32
	// maxADFNodes is the most nodes that are visited in one document.
	maxADFNodes = 200000
	// maxHeadingLevel is the deepest Markdown heading.
	maxHeadingLevel = 6
	// headingShift moves every heading down two levels, because the issue document already has a title and a Description heading.
	headingShift = 2

	truncatedMarker   = "[truncated]"
	unsupportedMarker = "[unsupported content]"
	removedLinkMarker = "[link removed]"
	defaultMention    = "@unknown"
	noNameAttachment  = "file"
)

// languagePattern is the shape of a code block language that is kept. Any other language is dropped.
var languagePattern = regexp.MustCompile(`^[A-Za-z0-9+#_-]{1,32}$`)

// adfNode is one node of an ADF tree. Only the members that the renderer reads are named.
type adfNode struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	Attrs   map[string]any `json:"attrs"`
	Marks   []adfMark      `json:"marks"`
	Content []adfNode      `json:"content"`
}

// adfMark is one mark of a text node.
type adfMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs"`
}

// sink is a bounded output buffer. A write that does not fit is cut at a whole character, and a unit that does not fit is
// refused whole. Either way the sink is marked full. It also remembers whether the last byte written ends a line.
type sink struct {
	b     strings.Builder
	limit int
	full  bool
	last  byte
	wrote bool
}

// write appends s, cut at a whole character when it does not fit.
func (k *sink) write(s string) {
	if k.full || s == "" {
		return
	}
	if room := k.limit - k.b.Len(); len(s) > room {
		cut := room
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut]
		k.full = true
	}
	k.appendRaw(s)
}

// unit appends s only when all of it fits. Otherwise nothing is written and the sink is marked full.
func (k *sink) unit(s string) {
	if k.full || s == "" {
		return
	}
	if len(s) > k.limit-k.b.Len() {
		k.full = true
		return
	}
	k.appendRaw(s)
}

// appendRaw writes s and records its last byte.
func (k *sink) appendRaw(s string) {
	if s == "" {
		return
	}
	k.b.WriteString(s)
	k.last = s[len(s)-1]
	k.wrote = true
}

// lineStart reports whether the next byte written starts a line.
func (k *sink) lineStart() bool {
	return !k.wrote || k.last == '\n'
}

// renderer walks one ADF tree and writes Markdown into its current sink.
type renderer struct {
	out       *sink
	nodes     int
	truncated bool
}

// RenderADF returns the Markdown of one ADF value. A missing or null value, or a document with no text, returns "", and the
// caller writes its own placeholder. A plain JSON string is one escaped paragraph. Input that is not JSON, or not an object
// or string, returns domain.ErrUnexpectedResponse. Unknown nodes render their children, or the marker "[unsupported
// content]". A tree over the depth or node limit, or an output over MaxTextBytes, renders what fits and ends with
// "[truncated]". The result holds no control or bidirectional character.
func RenderADF(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return "", nil
	}
	root, err := parseADF(trimmed)
	if err != nil {
		return "", err
	}
	const suffix = "\n\n" + truncatedMarker
	r := &renderer{out: &sink{limit: MaxTextBytes - len(suffix)}}
	r.block(root, 1)
	body := strings.TrimRight(r.out.b.String(), "\n")
	if !r.truncated && !r.out.full {
		return stripControls(body), nil
	}
	if body == "" {
		return truncatedMarker, nil
	}
	return stripControls(body + suffix), nil
}

// parseADF decodes trimmed JSON as one ADF node. A JSON string becomes a document of one paragraph.
func parseADF(trimmed []byte) (adfNode, error) {
	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return adfNode{}, domain.ErrUnexpectedResponse
		}
		return adfNode{Type: "doc", Content: []adfNode{{Type: "paragraph", Content: []adfNode{{Type: "text", Text: text}}}}}, nil
	}
	if trimmed[0] != '{' {
		return adfNode{}, domain.ErrUnexpectedResponse
	}
	var root adfNode
	if err := json.Unmarshal(trimmed, &root); err != nil {
		return adfNode{}, domain.ErrUnexpectedResponse
	}
	return root, nil
}

// visit counts one node and reports whether it may be rendered. A node below the depth limit, past the node limit, or after
// the output is full is not rendered, and the result is marked truncated.
func (r *renderer) visit(depth int) bool {
	r.nodes++
	if depth > maxADFDepth || r.nodes > maxADFNodes || r.out.full {
		r.truncated = true
		return false
	}
	return true
}

// capture runs render into a child sink that holds at most the room left in the current sink, and returns its text.
func (r *renderer) capture(render func()) string {
	parent := r.out
	child := &sink{limit: parent.limit - parent.b.Len()}
	r.out = child
	render()
	r.out = parent
	if child.full {
		r.truncated = true
	}
	return child.b.String()
}

// blocks renders each node in order as a block.
func (r *renderer) blocks(nodes []adfNode, depth int) {
	for _, n := range nodes {
		r.block(n, depth)
	}
}

// block renders one node that stands on its own lines. Each block ends with a blank line.
func (r *renderer) block(n adfNode, depth int) {
	if !r.visit(depth) {
		return
	}
	switch n.Type {
	case "doc", "panel", "mediaSingle", "mediaGroup", "listItem", "tableCell", "tableHeader":
		r.blocks(n.Content, depth+1)
	case "paragraph":
		r.inlines(n.Content, depth+1)
		r.out.write("\n\n")
	case "heading":
		r.out.write(strings.Repeat("#", headingLevel(n.Attrs)) + " ")
		r.inlines(n.Content, depth+1)
		r.out.write("\n\n")
	case "bulletList":
		r.list(n.Content, depth+1, false, 0)
	case "orderedList":
		r.list(n.Content, depth+1, true, attrInt(n.Attrs, "order", 1))
	case "taskList":
		r.markedItems(n.Content, depth+1, taskMark)
	case "decisionList":
		r.markedItems(n.Content, depth+1, func(adfNode) string { return "- " })
	case "blockquote":
		body := r.capture(func() { r.blocks(n.Content, depth+1) })
		r.out.write(prefixLines(body, "> ", "> ") + "\n\n")
	case "codeBlock":
		r.codeBlock(n)
	case "rule":
		r.out.write("---\n\n")
	case "expand", "nestedExpand":
		if title := attrString(n.Attrs, "title"); title != "" {
			r.out.write("**" + escapeFrom(title, false) + "**\n\n")
		}
		r.blocks(n.Content, depth+1)
	case "table":
		r.table(n.Content, depth+1)
	case "media":
		r.out.write(attachmentText(n.Attrs) + "\n\n")
	case "blockCard", "embedCard":
		r.out.write(cardText(n.Attrs) + "\n\n")
	default:
		if isInlineType(n.Type) {
			r.inline(n, depth)
			if n.Type != "hardBreak" {
				r.out.write("\n\n")
			}
			return
		}
		if len(n.Content) == 0 {
			r.out.write(unsupportedMarker + "\n\n")
			return
		}
		r.blocks(n.Content, depth+1)
	}
}

// list renders the items of a bullet or ordered list. A nested list is indented under its item by prefixLines.
func (r *renderer) list(items []adfNode, depth int, ordered bool, start int) {
	for i, item := range items {
		if !r.visit(depth) {
			return
		}
		body := r.capture(func() { r.itemBody(item, depth+1) })
		first := "- "
		if ordered {
			first = strconv.Itoa(start+i) + ". "
		}
		if i > 0 {
			r.out.write("\n")
		}
		r.out.write(prefixLines(body, first, strings.Repeat(" ", len(first))))
	}
	r.out.write("\n\n")
}

// markedItems renders task or decision items, one line each, with the mark that mark returns for the item.
func (r *renderer) markedItems(items []adfNode, depth int, mark func(adfNode) string) {
	for i, item := range items {
		if !r.visit(depth) {
			return
		}
		body := r.capture(func() { r.inlines(item.Content, depth+1) })
		m := mark(item)
		if i > 0 {
			r.out.write("\n")
		}
		r.out.write(prefixLines(body, m, strings.Repeat(" ", len(m))))
	}
	r.out.write("\n\n")
}

// itemBody renders the content of one list item. A list item holds blocks; any other node renders as a block itself.
func (r *renderer) itemBody(item adfNode, depth int) {
	if item.Type == "listItem" {
		r.blocks(item.Content, depth)
		return
	}
	r.block(item, depth)
}

// taskMark returns the Markdown check box of a task item.
func taskMark(item adfNode) string {
	if attrString(item.Attrs, "state") == "DONE" {
		return "- [x] "
	}
	return "- [ ] "
}

// table renders the rows of a table. The first row is the header, and a separator row follows it.
func (r *renderer) table(rows []adfNode, depth int) {
	for i, row := range rows {
		if !r.visit(depth) {
			return
		}
		var cells []string
		for _, cell := range row.Content {
			text := r.capture(func() { r.blocks(cell.Content, depth+1) })
			cells = append(cells, flattenCell(text))
		}
		if len(cells) == 0 {
			continue
		}
		r.out.write("| " + strings.Join(cells, " | ") + " |\n")
		if i == 0 {
			r.out.write("|" + strings.Repeat(" --- |", len(cells)) + "\n")
		}
	}
	r.out.write("\n")
}

// flattenCell turns the Markdown of one cell into one line, with every run of white space as one space.
func flattenCell(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// codeBlock writes a fenced code block whose fence is longer than any backtick run in its code. A block that does not fit is
// dropped whole, so no fence is left open.
func (r *renderer) codeBlock(n adfNode) {
	var code strings.Builder
	for _, c := range n.Content {
		if c.Type == "text" {
			code.WriteString(c.Text)
		}
	}
	text := strings.TrimSuffix(stripControls(code.String()), "\n")
	fence := strings.Repeat("`", max(3, longestBacktickRun(text)+1))
	lang := attrString(n.Attrs, "language")
	if !languagePattern.MatchString(lang) {
		lang = ""
	}
	r.out.unit(fence + lang + "\n" + text + "\n" + fence + "\n\n")
}

// inlines renders each node of a line in order.
func (r *renderer) inlines(nodes []adfNode, depth int) {
	for _, n := range nodes {
		r.inline(n, depth)
	}
}

// inline renders one node within a line.
func (r *renderer) inline(n adfNode, depth int) {
	if !r.visit(depth) {
		return
	}
	switch n.Type {
	case "text":
		r.text(n)
	case "hardBreak":
		r.out.write("\\\n")
	case "mention":
		r.out.write(mentionText(n.Attrs))
	case "emoji":
		r.out.write(emojiText(n.Attrs))
	case "status":
		r.out.write("[" + escapeFrom(attrString(n.Attrs, "text"), false) + "]")
	case "date":
		r.out.write(dateText(n.Attrs))
	case "inlineCard":
		r.out.write(cardText(n.Attrs))
	case "mediaInline":
		r.out.write(attachmentText(n.Attrs))
	default:
		if len(n.Content) == 0 {
			r.out.write(unsupportedMarker)
			return
		}
		r.inlines(n.Content, depth+1)
	}
}

// text writes one text node. A code mark makes the node a code span, and no other mark applies to it. Strong, emphasis,
// strike-through, and the first safe link wrap the escaped text. The wrapper parts are written as units, so a link never
// lacks its target when the output is cut.
func (r *renderer) text(n adfNode) {
	for _, m := range n.Marks {
		if m.Type == "code" {
			r.out.unit(codeSpan(stripControls(n.Text)))
			return
		}
	}
	var prefix, suffix string
	linked := false
	for _, m := range n.Marks {
		switch m.Type {
		case "strong":
			prefix += "**"
			suffix = "**" + suffix
		case "em":
			prefix += "*"
			suffix = "*" + suffix
		case "strike":
			prefix += "~~"
			suffix = "~~" + suffix
		case "link":
			href := attrString(m.Attrs, "href")
			if !linked && safeURL(href) {
				linked = true
				prefix += "["
				suffix = "](<" + href + ">)" + suffix
			}
		}
	}
	r.out.unit(prefix)
	r.out.write(escapeFrom(n.Text, r.out.lineStart()))
	r.out.unit(suffix)
}

// codeSpan returns s as an inline code span. The fence is one backtick longer than any run inside s, and s is padded with a
// space when it starts or ends with a backtick.
func codeSpan(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	fence := strings.Repeat("`", longestBacktickRun(s)+1)
	pad := ""
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		pad = " "
	}
	return fence + pad + s + pad + fence
}

// longestBacktickRun returns the length of the longest run of backticks in s.
func longestBacktickRun(s string) int {
	best, run := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			run++
			best = max(best, run)
		} else {
			run = 0
		}
	}
	return best
}

// prefixLines puts first before the first line of body and rest before each later line. An empty line gets rest without its
// trailing spaces, so a quote line reads ">" and a list line is empty.
func prefixLines(body, first, rest string) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		prefix := rest
		if i == 0 {
			prefix = first
		}
		if line == "" {
			b.WriteString(strings.TrimRight(prefix, " "))
		} else {
			b.WriteString(prefix + line)
		}
	}
	return b.String()
}

// isInlineType reports whether a node type is written within a line.
func isInlineType(t string) bool {
	switch t {
	case "text", "hardBreak", "mention", "emoji", "status", "date", "inlineCard", "mediaInline":
		return true
	}
	return false
}

// headingLevel returns the Markdown level of a heading after the shift, between 1 and maxHeadingLevel.
func headingLevel(attrs map[string]any) int {
	level := attrInt(attrs, "level", 1) + headingShift
	return min(max(level, 1), maxHeadingLevel)
}

// mentionText returns the escaped "@" text of a mention. The account ID is never read.
func mentionText(attrs map[string]any) string {
	text := attrString(attrs, "text")
	if text == "" {
		text = defaultMention
	}
	if !strings.HasPrefix(text, "@") {
		text = "@" + text
	}
	return escapeFrom(text, false)
}

// emojiText returns the escaped text of an emoji, or its short name in colons when it has no text.
func emojiText(attrs map[string]any) string {
	if text := attrString(attrs, "text"); text != "" {
		return escapeFrom(text, false)
	}
	short := strings.Trim(attrString(attrs, "shortName"), ":")
	if short == "" {
		return unsupportedMarker
	}
	return escapeFrom(":"+short+":", false)
}

// dateText returns the UTC day of an epoch-millisecond timestamp, as YYYY-MM-DD.
func dateText(attrs map[string]any) string {
	ms, ok := epochMillis(attrs["timestamp"])
	if !ok {
		return unsupportedMarker
	}
	return time.UnixMilli(ms).UTC().Format("2006-01-02")
}

// epochMillis reads a timestamp that Atlassian sends as a string or a number.
func epochMillis(v any) (int64, bool) {
	switch t := v.(type) {
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || t != math.Trunc(t) {
			return 0, false
		}
		return int64(t), true
	}
	return 0, false
}

// cardText returns a card as an autolink of its URL, or the removed-link marker when the URL is not a safe link.
func cardText(attrs map[string]any) string {
	href := attrString(attrs, "url")
	if !safeURL(href) {
		return removedLinkMarker
	}
	return "<" + href + ">"
}

// attachmentText names a media item by its alt text. No URL, ID, or token of the media is written.
func attachmentText(attrs map[string]any) string {
	name := attrString(attrs, "alt")
	if name == "" {
		name = noNameAttachment
	}
	return "[attachment: " + escapeFrom(name, false) + "]"
}

// attrString returns a string attribute, or "" when it is absent or not a string.
func attrString(attrs map[string]any, key string) string {
	s, _ := attrs[key].(string)
	return s
}

// attrInt returns a numeric attribute as an int between 0 and 1<<30, or def when it is absent or not a number.
func attrInt(attrs map[string]any, key string, def int) int {
	f, ok := attrs[key].(float64)
	if !ok || math.IsNaN(f) {
		return def
	}
	return int(math.Min(math.Max(f, 0), 1<<30))
}

// adfConvert maps one reply value through the ADF converter: the value is re-encoded to JSON and rendered to Markdown.
// A missing or null value maps to null, like any other missing optional field. A value that is not an ADF document or
// string is an unexpected response. The Markdown is charged to the shared output budget, like any mapped text.
func adfConvert(raw any, budget *int) (any, error) {
	if raw == nil {
		return nil, nil
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return nil, domain.ErrUnexpectedResponse
	}
	out, err := RenderADF(json.RawMessage(body))
	if err != nil {
		return nil, err
	}
	if len(out) > *budget {
		return nil, domain.ErrReplyTooLarge
	}
	*budget -= len(out)
	return out, nil
}

// maxLinkBytes is the longest URL that may be written as a link.
const maxLinkBytes = 2048

// markdownPunctuation holds the marks that escapeFrom backslash-escapes wherever they appear. Each one can start Markdown
// structure, a link, an image, an HTML tag, an entity, a code span, a strike-through, or a table cell break.
const markdownPunctuation = "\\\\`*_[]<>#|~!&"

// stripControls removes the characters a reviewer cannot see, or that reorder what is read: C0 controls except newline and
// tab, DEL, C1 controls, zero-width characters, bidirectional controls, and the byte order mark.
func stripControls(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 0x20 || r == 0x7f:
			return -1
		case r >= 0x80 && r <= 0x9f:
			return -1
		case r >= 0x200b && r <= 0x200f:
			return -1
		case r >= 0x202a && r <= 0x202e:
			return -1
		case r >= 0x2066 && r <= 0x2069:
			return -1
		case r == 0xfeff:
			return -1
		}
		return r
	}, s)
}

// escapeFrom returns s as Markdown text that holds no structure. It removes the hidden characters of stripControls, and it
// backslash-escapes the marks of markdownPunctuation. When lineStart is true, the text begins a line, so it escapes a "-",
// "+", or "=" that starts the line and the "." or ")" after a number, and it drops leading spaces and tabs so the line cannot
// open an indented code block. A line break is kept.
func escapeFrom(s string, lineStart bool) string {
	s = stripControls(s)
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\n':
			b.WriteByte(c)
			lineStart = true
			i++
			continue
		case lineStart && (c == ' ' || c == '\t'):
			i++
			continue
		case lineStart && (c == '-' || c == '+' || c == '='):
			b.WriteByte('\\')
			b.WriteByte(c)
			i++
		case lineStart && isDigit(c):
			j := i
			for j < len(s) && isDigit(s[j]) {
				j++
			}
			b.WriteString(s[i:j])
			if j < len(s) && (s[j] == '.' || s[j] == ')') {
				b.WriteByte('\\')
			}
			i = j
		case strings.IndexByte(markdownPunctuation, c) >= 0:
			b.WriteByte('\\')
			b.WriteByte(c)
			i++
		default:
			_, size := utf8.DecodeRuneInString(s[i:])
			b.WriteString(s[i : i+size])
			i += size
		}
		lineStart = false
	}
	return b.String()
}

// isDigit reports whether c is an ASCII decimal digit.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// safeURL reports whether raw may be written as a link. It must be an http, https, or mailto URL of at most maxLinkBytes
// bytes, with no space, control, bidirectional, or zero-width character, and no "<" or ">". Retriever never fetches a link.
func safeURL(raw string) bool {
	if raw == "" || len(raw) > maxLinkBytes {
		return false
	}
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "mailto:") {
		return false
	}
	if stripControls(raw) != raw {
		return false
	}
	for _, r := range raw {
		if r == '<' || r == '>' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}
