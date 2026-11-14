package cli

import (
	"bytes"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// TestANoticeRowPrintsItsNoticeInPlaceOfTheOffset checks that a truncated list with no offset to go on prints the row's
// notice, and not the next-page hint.
func TestANoticeRowPrintsItsNoticeInPlaceOfTheOffset(t *testing.T) {
	row, paged := pagedDocuments(t, domain.NewPage(0, 1, 1))
	row.Notice = "The list is truncated; narrow the query."
	var b bytes.Buffer
	writeContinuation(&b, row, paged)
	if b.String() != "The list is truncated; narrow the query.\n" {
		t.Fatalf("a notice row must print its notice, got %q", b.String())
	}
}

// TestAQuietEmptyListPrintsNothing checks that a quiet row prints neither its count line nor its rows for an empty list.
func TestAQuietEmptyListPrintsNothing(t *testing.T) {
	var b bytes.Buffer
	writeRows(&b, operations.Row{Quiet: true, Noun: "comments", Columns: []string{"id"}}, nil)
	if b.Len() != 0 {
		t.Fatalf("a quiet empty list must print nothing, got %q", b.String())
	}
}

// TestAnEmptyDetailFieldPrintsNothing checks that a row whose detail field is empty adds no line under the row.
func TestAnEmptyDetailFieldPrintsNothing(t *testing.T) {
	var b bytes.Buffer
	writeDetail(&b, domain.NewRecord(domain.Field{Name: "body", Value: ""}), "body")
	if b.Len() != 0 {
		t.Fatalf("an empty detail must print nothing, got %q", b.String())
	}
}

// TestAJSONTruncatedFlagFollowsTheNotice checks that a truncated list whose row has a notice reports truncated, and not
// has_more, because the notice stands in for the next offset.
func TestAJSONTruncatedFlagFollowsTheNotice(t *testing.T) {
	row, paged := pagedDocuments(t, domain.NewPage(0, 1, 1))
	row.Notice = "The list is truncated; narrow the query."
	doc := jsonDocument(row, paged)
	if doc["truncated"] != true {
		t.Fatalf("a notice row must report truncated, got %v", doc)
	}
	if _, ok := doc["has_more"]; ok {
		t.Fatalf("a notice row must not report has_more, got %v", doc)
	}
}

// TestANestedRecordPrintsAsANestedJSONObject checks that a record inside a field is written as a JSON object, so the JSON
// keeps the structure that the text form flattens.
func TestANestedRecordPrintsAsANestedJSONObject(t *testing.T) {
	user := domain.NewRecord(domain.Field{Name: "name", Value: "Ada"})
	out := recordMap(domain.NewRecord(domain.Field{Name: "user", Value: user}))
	nested, ok := out["user"].(map[string]any)
	if !ok || nested["name"] != "Ada" {
		t.Fatalf("a nested record must be a nested object, got %#v", out)
	}
}

// TestABodyWithoutALineFeedEndsWithOne checks that a body row prints its text on a line of its own.
func TestABodyWithoutALineFeedEndsWithOne(t *testing.T) {
	var b bytes.Buffer
	writeText(&b, operations.Row{Body: "text"}, domain.NewRecord(domain.Field{Name: "text", Value: "# Plan"}))
	if b.String() != "# Plan\n" {
		t.Fatalf("a body must end with a line feed, got %q", b.String())
	}
}

// TestARowDetailPrintsIndentedUnderItsLine checks that a detail field prints each of its lines, indented, under the row.
func TestARowDetailPrintsIndentedUnderItsLine(t *testing.T) {
	row := operations.Row{Noun: "comments", Columns: []string{"id", "name"}, Detail: "body"}
	items := []domain.Record{domain.NewRecord(
		domain.Field{Name: "id", Value: "c1"},
		domain.Field{Name: "name", Value: "Ada"},
		domain.Field{Name: "body", Value: "first\nsecond\n"},
	)}
	var b bytes.Buffer
	writeRows(&b, row, items)
	if want := "1 comment\n  c1  Ada\n    first\n    second\n"; b.String() != want {
		t.Fatalf("a row with a detail must print the detail under it, got %q, want %q", b.String(), want)
	}
}

// TestScalarTextWritesEachValueKind checks the text of the value kinds a record can hold: an int and a bool read as their
// digits and words, and any other kind reads through its default text, with its terminal controls replaced.
func TestScalarTextWritesEachValueKind(t *testing.T) {
	cases := []struct {
		value any
		want  string
	}{
		{42, "42"},
		{true, "true"},
		{1.5, "1.5"},
		{[]string{"a", "b"}, "[a b]"},
		{[]string{"a\x07"}, "[a�]"},
	}
	for _, c := range cases {
		if got := scalarText(c.value); got != c.want {
			t.Errorf("scalarText(%#v) = %q, want %q", c.value, got, c.want)
		}
	}
}
