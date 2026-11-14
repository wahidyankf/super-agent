package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// TestS9_ADepthAboveTheCapIsUnexpectedAndTheCapItselfPasses checks the nesting cap, one level at a time.
func TestS9_ADepthAboveTheCapIsUnexpectedAndTheCapItselfPasses(t *testing.T) {
	if _, err := decodeReplyWith([]byte(`[[[1]]]`), 3, 1000); err != nil {
		t.Fatalf("a depth of 3 at a cap of 3 was refused: %v", err)
	}
	if _, err := decodeReplyWith([]byte(`[[[[1]]]]`), 3, 1000); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("a depth of 4 at a cap of 3 = %v, want ErrUnexpectedResponse", err)
	}
}

// TestS9_ANodeAboveTheCapIsUnexpected checks the token cap. A list of one number has three tokens: the brackets and the
// number.
func TestS9_ANodeAboveTheCapIsUnexpected(t *testing.T) {
	if _, err := decodeReplyWith([]byte(`[1]`), 64, 3); err != nil {
		t.Fatalf("three tokens at a cap of 3 were refused: %v", err)
	}
	if _, err := decodeReplyWith([]byte(`[1,2]`), 64, 3); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("four tokens at a cap of 3 = %v, want ErrUnexpectedResponse", err)
	}
}

// TestS9_OnlyOneWholeJSONValueIsAReply checks the shapes a reply may not take.
func TestS9_OnlyOneWholeJSONValueIsAReply(t *testing.T) {
	for _, bad := range []string{``, `{"a":1} {"b":2}`, `[1`, `{"a":}`, `]`, `{"a":1}x`} {
		if _, err := decodeReply([]byte(bad)); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("decodeReply(%q) = %v, want ErrUnexpectedResponse", bad, err)
		}
	}
	v, err := decodeReply([]byte(` {"a": 1.5} `))
	if err != nil {
		t.Fatalf("a valid object was refused: %v", err)
	}
	if _, ok := v.(map[string]any)["a"].(json.Number); !ok {
		t.Fatal("a number was not kept as a json.Number")
	}
}

// TestPointerValueFollowsRFC6901 checks the JSON pointers that a spec may name.
func TestPointerValueFollowsRFC6901(t *testing.T) {
	doc := map[string]any{
		"a/b": map[string]any{"c~d": []any{"zero", "one"}},
		"list": []any{
			map[string]any{"x": "first"},
		},
		"plain": "value",
	}
	cases := []struct {
		name    string
		pointer string
		want    any
		found   bool
	}{
		{"the whole document", "", doc, true},
		{"an escaped slash and a tilde", "/a~1b/c~0d/1", "one", true},
		{"a list item then a key", "/list/0/x", "first", true},
		{"a missing key", "/missing", nil, false},
		{"a missing list item", "/list/9", nil, false},
		{"a leading zero is not an index", "/list/00", nil, false},
		{"a sign is not an index", "/list/+0", nil, false},
		{"a key of a scalar", "/plain/x", nil, false},
		{"a non-numeric index", "/list/first", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, found := pointerValue(doc, c.pointer)
			if found != c.found {
				t.Fatalf("found = %v, want %v", found, c.found)
			}
			if c.found && fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Fatalf("value = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCleanTextRemovesControlsAndKeepsLineBreaks checks what leaves a mapped text, and the budget that bounds it.
func TestCleanTextRemovesControlsAndKeepsLineBreaks(t *testing.T) {
	budget := 100
	got, err := cleanText("a\x07b\tc\nd\x00", &budget)
	if err != nil {
		t.Fatalf("cleanText: %v", err)
	}
	if got != "ab\tc\nd" {
		t.Fatalf("cleanText = %q, want the controls removed and tab and line feed kept", got)
	}
	if budget != 100-len(got) {
		t.Fatalf("budget = %d, want %d", budget, 100-len(got))
	}
}

// TestCleanTextFailsWhenTheOutputBudgetIsSpent checks the output cap.
func TestCleanTextFailsWhenTheOutputBudgetIsSpent(t *testing.T) {
	budget := 2
	if _, err := cleanText("abc", &budget); !errors.Is(err, domain.ErrReplyTooLarge) {
		t.Fatalf("cleanText over the budget = %v, want ErrReplyTooLarge", err)
	}
}

// TestScalarOfMapsEachJSONScalar checks the Go value of each JSON scalar, and refuses an object or a list.
func TestScalarOfMapsEachJSONScalar(t *testing.T) {
	budget := 1000
	cases := []struct {
		name string
		in   any
		want any
	}{
		{"null", nil, nil},
		{"true", true, true},
		{"a text", "hello", "hello"},
		{"a whole number", json.Number("42"), 42},
		{"a fraction is kept as text", json.Number("1.5"), "1.5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := scalarOf(c.in, &budget)
			if err != nil || got != c.want {
				t.Fatalf("scalarOf = %v, %v; want %v", got, err, c.want)
			}
		})
	}
	for _, bad := range []any{map[string]any{}, []any{}} {
		if _, err := scalarOf(bad, &budget); !errors.Is(err, domain.ErrUnexpectedResponse) {
			t.Fatalf("scalarOf(%T) = %v, want ErrUnexpectedResponse", bad, err)
		}
	}
}

// TestListItemsNeedsAListOfObjects checks the list shape that a list read maps.
func TestListItemsNeedsAListOfObjects(t *testing.T) {
	o := spec.Operation{Response: spec.Response{
		Items:  "/items",
		Fields: map[string]spec.Field{"id": {Pointer: "/id"}},
	}}
	budget := 1000
	if _, err := listItems(o, map[string]any{"items": "text"}, &budget); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("items that is text = %v, want ErrUnexpectedResponse", err)
	}
	if _, err := listItems(o, map[string]any{"items": []any{"x"}}, &budget); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("an item that is text = %v, want ErrUnexpectedResponse", err)
	}
	if _, err := listItems(o, map[string]any{}, &budget); !errors.Is(err, domain.ErrUnexpectedResponse) {
		t.Fatalf("a missing list = %v, want ErrUnexpectedResponse", err)
	}
	items, err := listItems(o, map[string]any{"items": []any{map[string]any{"id": "doc-1"}}}, &budget)
	if err != nil || len(items) != 1 || items[0].Str("id") != "doc-1" {
		t.Fatalf("listItems = %v, %v; want one document", items, err)
	}
}

// TestPlainValueReadsWholeNumbersAsIntegers checks the conversion of a parameter default or example value.
func TestPlainValueReadsWholeNumbersAsIntegers(t *testing.T) {
	if got := plainValue(json.Number("20")); got != 20 {
		t.Fatalf("plainValue(20) = %v, want the integer 20", got)
	}
	if got := plainValue(json.Number("2.5")); got != "2.5" {
		t.Fatalf("plainValue(2.5) = %v, want the text 2.5", got)
	}
	if got := plainValue("text"); got != "text" {
		t.Fatalf("plainValue(text) = %v", got)
	}
}

// TestDecodeReplyKeepsNumbersExact checks that a number is decoded without loss.
func TestDecodeReplyKeepsNumbersExact(t *testing.T) {
	v, err := decodeReply([]byte(`{"n": 12345678901234567890}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := v.(map[string]any)["n"].(json.Number).String(); got != "12345678901234567890" {
		t.Fatalf("number = %s, want it exact", got)
	}
}
