package jira

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxLinkBytes is the longest URL that may be written as a link.
const maxLinkBytes = 2048

// markdownPunctuation holds the marks that escapeText backslash-escapes wherever they appear. Each one can start Markdown
// structure, a link, an image, an HTML tag, an entity, a code span, a strike-through, or a table cell break.
const markdownPunctuation = "\\`*_[]<>#|~!&"

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

// escapeText returns s as Markdown text that starts a line. See escapeFrom.
func escapeText(s string) string {
	return escapeFrom(s, true)
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
