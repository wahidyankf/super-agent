package engine

import (
	"errors"
	"net/url"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// TestS7_APathSlotFillsOneWholeSegment checks the path rule: a slot is a whole segment, and a value that would leave its
// segment is refused.
func TestS7_APathSlotFillsOneWholeSegment(t *testing.T) {
	got, err := renderPath("/docs/{id}", map[string]any{"id": "doc-1"})
	if err != nil || got != "/docs/doc-1" {
		t.Fatalf("renderPath = %q, %v; want /docs/doc-1", got, err)
	}
	if _, err := renderPath("/docs/x{id}", map[string]any{"id": "doc-1"}); err == nil {
		t.Fatal("a slot that is part of a segment was rendered")
	}
	if _, err := renderPath("/docs/{id}", map[string]any{}); !errors.Is(err, domain.ErrMissingArgument) {
		t.Fatalf("a missing slot value = %v, want ErrMissingArgument", err)
	}
	for _, bad := range []string{"", ".", "..", "a/b", "a\\b", "a%2Fb", "a%", "doc\n1", "doc\x00"} {
		if _, err := renderPath("/docs/{id}", map[string]any{"id": bad}); !errors.Is(err, domain.ErrInvalidID) {
			t.Fatalf("renderPath with %q = %v, want ErrInvalidID", bad, err)
		}
	}
	if got, err := renderPath("/docs/{id}", map[string]any{"id": "doc 1"}); err != nil || got != "/docs/doc%201" {
		t.Fatalf("renderPath with a space = %q, %v; want the segment escaped", got, err)
	}
}

// TestS6_AHeaderSlotIsRefusedWithAControlCharacter checks the header rule: a constant is sent as it is, a slot is
// filled with its value, an unset slot is left out, and a value with a control character is refused.
func TestS6_AHeaderSlotIsRefusedWithAControlCharacter(t *testing.T) {
	headers := map[string]string{"X-Constant": "fixed", "X-Note": "{note}", "X-Unset": "{missing}"}
	h, err := renderHeaders(headers, map[string]any{"note": "first"})
	if err != nil {
		t.Fatalf("renderHeaders: %v", err)
	}
	if h.Get("X-Constant") != "fixed" || h.Get("X-Note") != "first" || h.Get("X-Unset") != "" {
		t.Fatalf("headers = %v", h)
	}
	if _, present := h["X-Unset"]; present {
		t.Fatal("an unset slot left a header behind")
	}
	for _, bad := range []string{"a\r\nSet-Cookie: x", "a\nb", "a\x00b", "a\x7fb", "a\xffb"} {
		if _, err := renderHeaders(map[string]string{"X-Note": "{note}"}, map[string]any{"note": bad}); !errors.Is(err, ErrInvalidHeader) {
			t.Fatalf("renderHeaders with %q = %v, want ErrInvalidHeader", bad, err)
		}
	}
	if _, err := renderHeaders(map[string]string{"X-Note": "{note}"}, map[string]any{"note": "tab\tis fine"}); err != nil {
		t.Fatalf("a tab in a header value was refused: %v", err)
	}
}

// TestS7_ABodyValueIsJSONEncodedAndAPartialSlotStaysText checks the body rule: a whole slot takes a typed value, an
// absent optional value leaves its key out, and a slot inside a longer text is only text.
func TestS7_ABodyValueIsJSONEncodedAndAPartialSlotStaysText(t *testing.T) {
	o := spec.Operation{Body: map[string]any{
		"title":  "{{title}}",
		"text":   "{{text}}",
		"note":   "x {{title}}",
		"nested": map[string]any{"inner": "{{title}}"},
		"items":  []any{"{{title}}", 1},
	}}
	raw, err := renderBody(o, map[string]any{"title": "a \"quoted\"\r\nline"})
	if err != nil {
		t.Fatalf("renderBody: %v", err)
	}
	want := `{"items":["a \"quoted\"\r\nline",1],"nested":{"inner":"a \"quoted\"\r\nline"},"note":"x {{title}}","title":"a \"quoted\"\r\nline"}`
	if string(raw) != want {
		t.Fatalf("body = %s\nwant %s", raw, want)
	}
	if none, err := renderBody(spec.Operation{}, nil); err != nil || none != nil {
		t.Fatalf("an operation with no body = %q, %v; want nothing", none, err)
	}
}

// TestRenderRequestPutsQueryValuesInTheQueryAndPagingOverrides checks that a declared query value is encoded and that a
// paging value replaces the declared one.
func TestRenderRequestPutsQueryValuesInTheQueryAndPagingOverrides(t *testing.T) {
	o := spec.Operation{
		Method: "GET",
		Path:   "/docs",
		Params: []spec.Param{{Name: "limit", Type: "Limit", In: "query"}, {Name: "q", Type: "Text", In: "query"}},
	}
	args := domain.NewArgs(map[string]any{"limit": 20, "q": "a&b"})
	req, err := renderRequest("/v1", o, args, url.Values{"cursor": {"c2"}, "limit": {"5"}})
	if err != nil {
		t.Fatalf("renderRequest: %v", err)
	}
	if req.target != "/v1/docs?cursor=c2&limit=5&q=a%26b" {
		t.Fatalf("target = %q", req.target)
	}
}

// TestTextOfCoversEachArgumentType checks the text that a value takes in a query or a header.
func TestTextOfCoversEachArgumentType(t *testing.T) {
	cases := map[string]struct {
		in   any
		want string
	}{
		"text":    {"x", "x"},
		"integer": {7, "7"},
		"true":    {true, "true"},
		"list":    {[]string{"a", "b"}, "a,b"},
		"other":   {3.5, "3.5"},
	}
	for name, c := range cases {
		if got := textOf(c.in); got != c.want {
			t.Fatalf("%s: textOf = %q, want %q", name, got, c.want)
		}
	}
}

// TestHeaderSafetyRefusesControlsAndAcceptsText checks the header value test on its own.
func TestHeaderSafetyRefusesControlsAndAcceptsText(t *testing.T) {
	if !headerSafe("plain text 1-2_3") {
		t.Fatal("plain text was refused")
	}
	if headerSafe("line\nbreak") {
		t.Fatal("a line break was accepted")
	}
}
