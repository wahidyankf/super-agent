package jira

import (
	"strings"
	"testing"
)

// TestStripControlsRemovesEachHiddenClass checks each class of character that stripControls removes, and the ones it keeps.
func TestStripControlsRemovesEachHiddenClass(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "C0 control", in: "a\x01b", want: "ab"},
		{name: "escape byte", in: "a\x1bb", want: "ab"},
		{name: "DEL", in: "a\x7fb", want: "ab"},
		{name: "C1 control", in: "a\u0085b", want: "ab"},
		{name: "zero-width space", in: "a\u200Bb", want: "ab"},
		{name: "zero-width joiner", in: "a\u200Db", want: "ab"},
		{name: "left-to-right mark", in: "a\u200Eb", want: "ab"},
		{name: "right-to-left override", in: "a\u202Eb", want: "ab"},
		{name: "isolate", in: "a\u2066b", want: "ab"},
		{name: "byte order mark", in: "a\uFEFFb", want: "ab"},
		{name: "newline is kept", in: "a\nb", want: "a\nb"},
		{name: "tab is kept", in: "a\tb", want: "a\tb"},
		{name: "ordinary Unicode is kept", in: "héllo \u263a 日本", want: "héllo \u263a 日本"},
		{name: "empty stays empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripControls(tt.in); got != tt.want {
				t.Fatalf("stripControls(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestEscapeTextEscapesEveryMarkdownPunctuationMark checks each punctuation mark that escapeText backslash-escapes.
func TestEscapeTextEscapesEveryMarkdownPunctuationMark(t *testing.T) {
	for _, mark := range []string{"\\", "`", "*", "_", "[", "]", "<", ">", "#", "|", "~", "!", "&"} {
		got := escapeText("a" + mark + "b")
		if got != "a\\"+mark+"b" {
			t.Errorf("escapeText(a%sb) = %q, want the mark escaped", mark, got)
		}
	}
}

// TestEscapeTextMarksOnlyAtLineStart checks the line-start markers, which are escaped only at the start of a line.
func TestEscapeTextMarksOnlyAtLineStart(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "minus inside a line is plain", in: "a - b", want: "a - b"},
		{name: "plus inside a line is plain", in: "a + b", want: "a + b"},
		{name: "minus after a line break is escaped", in: "a\n- b", want: "a\n\\- b"},
		{name: "digits then a dot after a break are escaped", in: "a\n10. b", want: "a\n10\\. b"},
		{name: "digits then a dot inside a line are plain", in: "a 10. b", want: "a 10. b"},
		{name: "digits alone at line start are plain", in: "10 apples", want: "10 apples"},
		{name: "spaces at line start are dropped", in: "a\n   b", want: "a\nb"},
		{name: "empty text", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeText(tt.in); got != tt.want {
				t.Fatalf("escapeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestSafeURLAcceptsOnlyWebAndMailLinks checks the scheme and character rules of a link.
func TestSafeURLAcceptsOnlyWebAndMailLinks(t *testing.T) {
	ok := []string{
		"https://example.com/a?b=c",
		"http://example.com",
		"HTTPS://EXAMPLE.COM",
		"mailto:ann@example.com",
		"https://example.com/" + strings.Repeat("a", maxLinkBytes-len("https://example.com/")),
	}
	for _, raw := range ok {
		if !safeURL(raw) {
			t.Errorf("safeURL(%q) = false, want true", raw)
		}
	}
	bad := []string{
		"",
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html,<b>x</b>",
		"vbscript:msgbox",
		"file:///etc/passwd",
		"ftp://example.com",
		"//example.com",
		"example.com",
		"https://example.com/a b",
		"https://example.com/\tb",
		"https://example.com/\nb",
		"https://example.com/<x>",
		"https://example.com/\u200Bb",
		"https://example.com/\u202Eb",
		"https://example.com/\u00A0b",
		"https://example.com/" + strings.Repeat("a", maxLinkBytes),
	}
	for _, raw := range bad {
		if safeURL(raw) {
			t.Errorf("safeURL(%q) = true, want false", raw)
		}
	}
}
