package cli

import (
	"bytes"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
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
