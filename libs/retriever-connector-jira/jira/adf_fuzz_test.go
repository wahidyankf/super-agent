package jira

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// FuzzRenderADF checks RenderADF on arbitrary input. The seed corpus runs on every go test. A run of the fuzzer
// searches for input that panics, that breaks the output cap, that writes a control or bidirectional character, or that
// writes a bare "<" outside a link, an autolink, or a code span or fence.
func FuzzRenderADF(f *testing.F) {
	seeds := []string{
		string(adfDoc(para("<b>x</b> # y"))),
		string(adfDoc(`{"type":"codeBlock","content":[{"type":"text","text":"<x>"}]}`)),
		string(adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"l","marks":[{"type":"link","attrs":{"href":"https://e.x/<a>"}}]}]}`)),
		string(adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"c<d","marks":[{"type":"code"}]}]}`)),
		string(adfDoc(`{"type":"inlineCard","attrs":{"url":"https://e.x"}}`)),
		string(adfDoc(`{"type":"table","content":[{"type":"tableRow","content":[{"type":"tableCell","content":[` + para("a|<b>") + `]}]}]}`)),
		string(adfDoc(`{"type":"bulletList","content":[{"type":"listItem","content":[` + para("1. <x>") + `]}]}`)),
		string(adfDoc(`{"type":"paragraph","content":[{"type":"text","text":"\u202Ehidden\u200B"}]}`)),
		string(nestedPanels(40, "deep")),
		`"plain <i>"`,
		`null`,
		`{"type":`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		out, err := RenderADF(json.RawMessage(raw))
		if err != nil {
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("RenderADF error = %v, want ErrUnexpectedResponse", err)
			}
			return
		}
		checkADFOutput(t, out)
	})
}

// checkADFOutput asserts the output properties that every rendered description must hold.
func checkADFOutput(t *testing.T, out string) {
	t.Helper()
	if len(out) > MaxTextBytes {
		t.Fatalf("output is %d bytes, over the %d cap", len(out), MaxTextBytes)
	}
	if !utf8.ValidString(out) {
		t.Fatalf("output is not valid UTF-8")
	}
	if stripControls(out) != out {
		t.Fatalf("output holds a control or bidirectional character: %q", out)
	}
	if hasBareAngle(out) {
		t.Fatalf("output holds a bare < outside a link, an autolink, or code: %q", out)
	}
}

// hasBareAngle reports whether out holds a "<" that Markdown would read as HTML. The scan skips escaped characters, fenced
// code, code spans, and autolinks of the form <target> with no space or "<" inside.
func hasBareAngle(out string) bool {
	lines := strings.Split(out, "\n")
	inFence := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " >"), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if lineHasBareAngle(line) {
			return true
		}
	}
	return false
}

// lineHasBareAngle scans one line outside any fence. It returns true for a "<" that is not escaped, not inside a code
// span, and not the start of an autolink.
func lineHasBareAngle(line string) bool {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++ // the next byte is an escaped character
		case '`':
			run := backtickRun(line, i)
			end := strings.Index(line[i+run:], strings.Repeat("`", run))
			if end < 0 {
				i += run - 1
				continue
			}
			i += run + end + run - 1
		case '<':
			gt := strings.IndexByte(line[i:], '>')
			if gt < 0 {
				return true
			}
			target := line[i+1 : i+gt]
			if strings.ContainsAny(target, " <") {
				return true
			}
			i += gt
		}
	}
	return false
}

// backtickRun returns the number of consecutive backticks that start at index i.
func backtickRun(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}
