package engine

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// docOperation returns a hand-built read of one object with the document layout of the tests: the title, one line
// of each closed kind, and the description body. The pair reads its own fields, so a misshaped pair value fails at
// the pair line instead of an earlier line.
func docOperation() spec.Operation {
	return spec.Operation{Response: spec.Response{
		Fields: map[string]spec.Field{
			"title": {Pointer: "/title"},
			"key":   {Pointer: "/key"},
			"name":  {Pointer: "/name"},
			"pname": {Pointer: "/pname"},
			"pkey":  {Pointer: "/pkey"},
			"tags":  {Pointer: "/tags", Format: "strings"},
			"desc":  {Pointer: "/desc", Convert: "markdown-from-adf"},
		},
		Document: &spec.Document{
			Title: "title",
			Lines: []spec.DocLine{
				{Label: "Key", Verbatim: "key"},
				{Label: "Name", Field: "name", Default: "Nobody", SkipEmpty: true},
				{Label: "Pair", Pair: &spec.DocPair{Name: "pname", Key: "pkey"}},
				{Label: "Tags", Field: "tags", Join: ", ", SkipEmpty: true},
			},
			Body: spec.DocBody{Field: "desc", Convert: "markdown-from-adf", Heading: "Body", Empty: "(empty)"},
		},
	}}
}

// docRecord returns a valid record for the test layout.
func docRecord() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "title", Value: "T"},
		domain.Field{Name: "key", Value: "EXAMPLE-1"},
		domain.Field{Name: "name", Value: "Ann"},
		domain.Field{Name: "pname", Value: "P"},
		domain.Field{Name: "pkey", Value: "EXAMPLE-9"},
		domain.Field{Name: "tags", Value: []any{"a", "b"}},
		domain.Field{Name: "desc", Value: "D"})
}

// withValue returns the test record with one field set to the value.
func withValue(name string, value any) domain.Record {
	rec := docRecord()
	fields := rec.Fields()
	for i, f := range fields {
		if f.Name == name {
			fields[i].Value = value
		}
	}
	return domain.NewRecord(fields...)
}

// withoutValue returns the test record with one field dropped.
func withoutValue(name string) domain.Record {
	rec := docRecord()
	var fields []domain.Field
	for _, f := range rec.Fields() {
		if f.Name != name {
			fields = append(fields, f)
		}
	}
	return domain.NewRecord(fields...)
}

// TestEscapeTextEscapesEveryMarkdownPunctuationMark checks the moved escape: each mark that can start Markdown
// structure is backslash-escaped, exactly as the Jira connector escapes it.
func TestEscapeTextEscapesEveryMarkdownPunctuationMark(t *testing.T) {
	for _, mark := range []string{"\\", "`", "*", "_", "[", "]", "<", ">", "#", "|", "~", "!", "&"} {
		if got := escapeText("a" + mark + "b"); got != "a\\"+mark+"b" {
			t.Errorf("escapeText(a%sb) = %q, want the mark escaped", mark, got)
		}
	}
}

// TestEscapeTextMarksOnlyAtLineStart checks the line-start marks of the moved escape: a leading dash is escaped,
// and leading spaces are dropped so the value cannot open an indented code block.
func TestEscapeTextMarksOnlyAtLineStart(t *testing.T) {
	if got := escapeText("- item"); got != "\\- item" {
		t.Fatalf("escapeText(- item) = %q, want the escaped dash", got)
	}
	if got := escapeText("   indented"); got != "indented" {
		t.Fatalf("escapeText of leading spaces = %q, want them dropped", got)
	}
	if got := escapeText("10. numbered"); got != "10\\. numbered" {
		t.Fatalf("escapeText(10. numbered) = %q, want the escaped dot", got)
	}
}

// TestEscapeTextStripsHiddenCharacters checks the moved escape on hidden characters: controls, zero-width marks,
// and direction overrides are gone, and the text stays literal.
func TestEscapeTextStripsHiddenCharacters(t *testing.T) {
	if got := escapeText("a\x01b\u200bc\u202ed"); got != "abcd" {
		t.Fatalf("escapeText with hidden characters = %q, want them stripped", got)
	}
}

// TestRenderDocumentFailsOnAMissingTitle checks that a record without the title field fails: the title is never
// optional.
func TestRenderDocumentFailsOnAMissingTitle(t *testing.T) {
	o := docOperation()
	if _, err := renderDocument(o, domain.NewRecord(), intPtr(1<<20)); err == nil {
		t.Fatal("a document with no title was rendered")
	}
}

// TestRenderDocumentFailsOnAMisshapedValue checks that a non-text title, line, pair, or body value fails the
// read naming the field, never the value.
func TestRenderDocumentFailsOnAMisshapedValue(t *testing.T) {
	o := docOperation()
	cases := map[string]domain.Record{
		"title": withValue("title", 7),
		"name":  withValue("name", 7),
		"key":   withValue("key", 7),
		"pname": withValue("pname", 7),
		"pkey":  withValue("pkey", 7),
		"body":  withValue("desc", 7),
	}
	for name, rec := range cases {
		if _, err := renderDocument(o, rec, intPtr(1<<20)); err == nil {
			t.Fatalf("a document with a misshaped %s was rendered", name)
		} else if strings.Contains(err.Error(), "7") {
			t.Fatalf("the failure echoes the value: %q", err.Error())
		}
	}
}

// TestRenderDocumentJoinsOnlyAListOfStrings checks that a joined line fails on a value that is not a list of
// strings: a text, or a list with a non-text element.
func TestRenderDocumentJoinsOnlyAListOfStrings(t *testing.T) {
	o := docOperation()
	for _, tags := range []any{"oops", []any{"ok", 7}} {
		if _, err := renderDocument(o, withValue("tags", tags), intPtr(1<<20)); err == nil {
			t.Fatalf("a document with tags %v was rendered", tags)
		}
	}
}

// TestRenderDocumentRendersMissingVerbatimAsEmpty checks that a missing verbatim value is empty: the line is
// emitted with no value.
func TestRenderDocumentRendersMissingVerbatimAsEmpty(t *testing.T) {
	o := docOperation()
	got, err := renderDocument(o, withoutValue("key"), intPtr(1<<20))
	if err != nil {
		t.Fatalf("renderDocument() error = %v, want nil", err)
	}
	if md := got.Str("markdown"); !strings.Contains(md, "- Key: \n") {
		t.Fatalf("the key line is not empty:\n%s", md)
	}
}

// TestRenderDocumentRendersABlankBodyAsEmpty checks that a blank description renders the empty constant.
func TestRenderDocumentRendersABlankBodyAsEmpty(t *testing.T) {
	o := docOperation()
	got, err := renderDocument(o, withValue("desc", ""), intPtr(1<<20))
	if err != nil {
		t.Fatalf("renderDocument() error = %v, want nil", err)
	}
	if md := got.Str("markdown"); !strings.HasSuffix(md, "## Body\n\n(empty)") {
		t.Fatalf("the body is not the empty constant:\n%s", md)
	}
}

// TestRenderDocumentRendersAPairWithoutAName checks the bare-key pair: an empty name renders the key alone, with no
// parentheses.
func TestRenderDocumentRendersAPairWithoutAName(t *testing.T) {
	o := docOperation()
	got, err := renderDocument(o, withValue("pname", nil), intPtr(1<<20))
	if err != nil {
		t.Fatalf("renderDocument() error = %v, want nil", err)
	}
	if md := got.Str("markdown"); !strings.Contains(md, "- Pair: EXAMPLE-9\n") {
		t.Fatalf("the pair line is not the bare key:\n%s", md)
	}
}

// TestRenderDocumentOverBudgetIsTooLarge checks that a document past the shared budget fails instead of rendering.
func TestRenderDocumentOverBudgetIsTooLarge(t *testing.T) {
	o := docOperation()
	if _, err := renderDocument(o, withValue("title", "A title that does not fit"), intPtr(4)); err == nil {
		t.Fatal("a document past the budget was rendered")
	}
}
