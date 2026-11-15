package engine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// TestEditorConverterIsServed checks that the converter gate lets the editor-data converter through for reads,
// unlike the HTML converter it refuses before any request.
func TestEditorConverterIsServed(t *testing.T) {
	op := spec.Operation{Response: spec.Response{Fields: map[string]spec.Field{
		"text": {Pointer: "/text", Convert: "text-from-editor-data"},
	}}}
	if err := converterUnsupported(op); err != nil {
		t.Fatalf("converterUnsupported = %v, want nil", err)
	}
	if err := unsupported(op); err != nil {
		t.Fatalf("unsupported = %v, want nil", err)
	}
}

// editorValue decodes editor JSON for the converter tests. A value that is not JSON fails the test.
func editorValue(t *testing.T, raw string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("the editor JSON does not decode: %v", err)
	}
	return v
}

// TestEditorTextReadsEditorNodes checks the plain-text extraction of the editor JSON that a comment keeps: the text
// of each block on its own line, a hard break as a line break, a mention as its label with an at sign, and marks
// dropped. The cases mirror the Go projection's own table, so the converter reads what the gateway reads.
func TestEditorTextReadsEditorNodes(t *testing.T) {
	cases := []struct {
		name string
		node string
		want string
	}{
		{"one paragraph", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hi"}]}]}`, "Hi"},
		{"a bold word", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a "},` +
			`{"type":"text","text":"bold","marks":[{"type":"strong"}]}]}]}`, "a bold"},
		{"a hard break", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a"},` +
			`{"type":"hard_break"},{"type":"text","text":"b"}]}]}`, "a\nb"},
		{"a mention", `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi "},` +
			`{"type":"mention","attrs":{"label":"Ada"}}]}]}`, "hi @Ada"},
		{"a list of items", `{"type":"doc","content":[{"type":"bulletList","content":[{"type":"listItem","content":` +
			`[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},{"type":"listItem","content":` +
			`[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}]}]}`, "one\ntwo"},
		{"an empty paragraph", `{"type":"doc","content":[{"type":"paragraph"}]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			budget := 1 << 20
			got, err := editorText(editorValue(t, tc.node), &budget)
			if err != nil {
				t.Fatalf("editorText: %v", err)
			}
			if got != tc.want {
				t.Fatalf("editorText = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEditorTextPassesPlainTextThrough checks the fallback branch of the converter: a plain string is already text,
// so it passes through unchanged instead of being read as a node.
func TestEditorTextPassesPlainTextThrough(t *testing.T) {
	budget := 1 << 20
	got, err := editorText("Nice", &budget)
	if err != nil {
		t.Fatalf("editorText: %v", err)
	}
	if got != "Nice" {
		t.Fatalf("editorText = %q, want the text unchanged", got)
	}
}

// TestEditorTextMapsMissingToNull checks that a missing or null editor value maps to null, like any other missing
// optional field, and that no partial text is returned.
func TestEditorTextMapsMissingToNull(t *testing.T) {
	budget := 1 << 20
	for _, raw := range []any{nil} {
		got, err := editorText(raw, &budget)
		if err != nil || got != nil {
			t.Fatalf("editorText = %v, %v; want null", got, err)
		}
	}
}

// TestEditorTextRefusesAnotherScalar checks the fail-closed branch of the converter: a number or a flag is neither
// editor data nor text, so it is an unexpected response instead of an empty text.
func TestEditorTextRefusesAnotherScalar(t *testing.T) {
	budget := 1 << 20
	for _, raw := range []any{float64(3), true} {
		if _, err := editorText(raw, &budget); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("editorText(%v) = %v, want ErrUnexpectedResponse", raw, err)
		}
	}
}

// TestEditorTextSkipsWhatIsNotANode checks that a child that is not an editor node writes nothing, and that a mention
// with no label writes nothing either, so neither shape panics nor invents text.
func TestEditorTextSkipsWhatIsNotANode(t *testing.T) {
	budget := 1 << 20
	node := editorValue(t, `{"type":"doc","content":[{"type":"paragraph","content":["stray",42,`+
		`{"type":"text","text":"a"},{"type":"mention"},{"type":"mention","attrs":{}}]}]}`)
	got, err := editorText(node, &budget)
	if err != nil {
		t.Fatalf("editorText: %v", err)
	}
	if got != "a" {
		t.Fatalf("editorText = %q, want only the text node", got)
	}
}

// TestEditorTextFailsAnObjectPastTheBudget checks that editor text past the budget fails, so a reply cannot build
// more text than one reply may hold.
func TestEditorTextFailsAnObjectPastTheBudget(t *testing.T) {
	budget := 4
	node := editorValue(t, `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello"}]}]}`)
	if _, err := editorText(node, &budget); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("editorText = %v, want ErrReplyTooLarge", err)
	}
}

// TestEditorTextChargesTheSharedBudget checks that a text past the budget fails, so a reply cannot build more text
// than one reply may hold.
func TestEditorTextChargesTheSharedBudget(t *testing.T) {
	budget := 2
	if _, err := editorText("abc", &budget); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("an over-budget text passes")
	}
	budget = 64 << 10
	deep := `{"type":"doc","content":[`
	for i := 0; i < 70; i++ {
		deep += `{"type":"paragraph","content":[`
	}
	deep += `{"type":"text","text":"deep"}`
	deep += strings.Repeat(`]}`, 70) + `]}`
	got, err := editorText(editorValue(t, deep), &budget)
	if err != nil {
		t.Fatalf("editorText of a deep tree: %v", err)
	}
	if got != "" {
		t.Fatalf("editorText = %q, want nothing: nodes past the cap are dropped", got)
	}
}
