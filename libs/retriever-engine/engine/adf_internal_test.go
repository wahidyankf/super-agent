package engine

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestSinkWritePastAnExactlyFullSinkWritesNothing checks the bounded-buffer edge: a write that starts where the room
// ends is cut to nothing, marks the sink full, and leaves the bytes alone.
func TestSinkWritePastAnExactlyFullSinkWritesNothing(t *testing.T) {
	k := &sink{limit: 3}
	k.write("abc")
	k.write("d")
	if !k.full {
		t.Fatal("a write past an exactly full sink leaves it unmarked")
	}
	if got := k.b.String(); got != "abc" {
		t.Fatalf("bytes = %q, want %q", got, "abc")
	}
}

// TestRenderADFRefusesABrokenString checks that a JSON string that does not parse fails as an unexpected response,
// with no partial Markdown.
func TestRenderADFRefusesABrokenString(t *testing.T) {
	if _, err := RenderADF(json.RawMessage(`"abc`)); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestRenderADFCaptureOverflowTruncates checks that a captured render that does not fit marks the result truncated: a
// blockquote holding an oversized code block ends with the marker and leaves no fence open.
func TestRenderADFCaptureOverflowTruncates(t *testing.T) {
	code := strings.Repeat("x", MaxTextBytes)
	in := adfDoc(`{"type":"blockquote","content":[{"type":"codeBlock","content":[{"type":"text","text":` + jsonString(code) + `}]}]}`)
	got, err := RenderADF(in)
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if strings.Contains(got, "```") {
		t.Fatalf("a captured oversized code block left a fence open (tail %q)", tail(got))
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Fatalf("a captured overflow has no marker (tail %q)", tail(got))
	}
}

// TestRenderADFNodeCapAcrossListsTablesAndInlines checks that a node count exhausted inside a bullet list stops every
// later construct too: the task list, the table, and the trailing paragraph render nothing, the output ends with the
// marker, and nothing panics.
func TestRenderADFNodeCapAcrossListsTablesAndInlines(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"type":"doc","content":[`)
	b.WriteString(`{"type":"bulletList","content":[`)
	for i := 0; i < maxADFNodes+10; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"type":"listItem","content":[{"type":"rule"}]}`)
	}
	b.WriteString(`]},`)
	b.WriteString(`{"type":"taskList","content":[{"type":"taskItem","attrs":{"state":"DONE"},"content":[{"type":"text","text":"done"}]}]},`)
	b.WriteString(`{"type":"table","content":[{"type":"tableRow","content":[{"type":"tableCell","content":[` + para("cell") + `]}]}]},`)
	b.WriteString(para("after"))
	b.WriteString(`]}`)
	got, err := RenderADF(json.RawMessage(b.String()))
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	for _, hidden := range []string{"done", "cell", "after"} {
		if strings.Contains(got, hidden) {
			t.Fatalf("text past the node cap is shown: %q", hidden)
		}
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Fatalf("a tree over the node limit has no marker (tail %q)", tail(got))
	}
}

// TestRenderADFNodeCapEndsEveryRepeatingConstruct checks that a node count exhausted inside each repeating construct
// stops that construct: a task list, a table, and a paragraph of inline nodes each render what fits and end with the
// marker, and nothing panics.
func TestRenderADFNodeCapEndsEveryRepeatingConstruct(t *testing.T) {
	build := func(open, item, close string) json.RawMessage {
		var b strings.Builder
		b.WriteString(`{"type":"doc","content":[` + open)
		for i := 0; i < maxADFNodes+10; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(item)
		}
		b.WriteString(close + `]}`)
		return json.RawMessage(b.String())
	}
	docs := map[string]json.RawMessage{
		"task list": build(`{"type":"taskList","content":[`,
			`{"type":"taskItem","attrs":{"state":"DONE"},"content":[{"type":"text","text":"done"}]}`,
			`]}`), "table": build(`{"type":"table","content":[`,
			`{"type":"tableRow","content":[{"type":"tableCell","content":[`+para("cell")+`]}]}`,
			`]}`),
		"paragraph": build(`{"type":"paragraph","content":[`,
			`{"type":"text","text":"after"}`,
			`]}`),
	}
	names := []string{"task list", "table", "paragraph"}
	for _, name := range names {
		got, err := RenderADF(docs[name])
		if err != nil {
			t.Fatalf("%s: RenderADF: %v", name, err)
		}
		if !strings.HasSuffix(got, "[truncated]") {
			t.Fatalf("%s over the node limit has no marker (tail %q)", name, tail(got))
		}
		if len(got) > MaxTextBytes {
			t.Fatalf("%s output %d bytes is over the cap", name, len(got))
		}
	}
}

// TestRenderADFDateWithANonScalarTimestamp checks that a date whose timestamp is neither text nor a number renders the
// unsupported marker instead of a date.
func TestRenderADFDateWithANonScalarTimestamp(t *testing.T) {
	in := adfDoc(`{"type":"paragraph","content":[{"type":"date","attrs":{"timestamp":true}}]}`)
	got, err := RenderADF(in)
	if err != nil {
		t.Fatalf("RenderADF: %v", err)
	}
	if got != "[unsupported content]" {
		t.Fatalf("got = %q, want the unsupported marker", got)
	}
}

// TestADFConvertRefusesAnUnmarshalableValue checks that a reply value with no JSON shape fails as an unexpected
// response instead of reaching the renderer.
func TestADFConvertRefusesAnUnmarshalableValue(t *testing.T) {
	budget := 100
	if _, err := adfConvert(math.NaN(), &budget); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

// TestADFConvertBudgetExceeded checks that Markdown longer than the remaining output budget fails instead of growing
// past the reply cap.
func TestADFConvertBudgetExceeded(t *testing.T) {
	var raw any
	if err := json.Unmarshal(adfDoc(para("Hello")), &raw); err != nil {
		t.Fatalf("decode the fixture: %v", err)
	}
	budget := 2
	if _, err := adfConvert(raw, &budget); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("err = %v, want ErrReplyTooLarge", err)
	}
}

// TestStripControlsRemovesTheRemainingHiddenClasses checks the two hidden classes the converter tables do not reach:
// C1 controls and isolate controls.
func TestStripControlsRemovesTheRemainingHiddenClasses(t *testing.T) {
	if got := stripControls("a\u0080b\u009fc"); got != "abc" {
		t.Fatalf("C1 controls stay: %q", got)
	}
	if got := stripControls("a\u2066b\u2069c"); got != "abc" {
		t.Fatalf("isolate controls stay: %q", got)
	}
}

// TestSafeURLRefusesEmptyAndHiddenCharacters checks that an empty address and an address with a hidden character are
// never written as links.
func TestSafeURLRefusesEmptyAndHiddenCharacters(t *testing.T) {
	if safeURL("") {
		t.Fatal("an empty address is a safe URL")
	}
	if safeURL("https://e.x/\u200bok") {
		t.Fatal("an address with a hidden character is a safe URL")
	}
}
