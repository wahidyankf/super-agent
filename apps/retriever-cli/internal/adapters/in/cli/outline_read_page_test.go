package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// pagedDocuments returns the documents.list row and a list of one document that is paged at the given page.
func pagedDocuments(t *testing.T, page domain.Page) (operations.Row, domain.Record) {
	t.Helper()
	row, ok := operations.Lookup(domain.OpDocumentsList)
	if !ok {
		t.Fatal("no documents.list row")
	}
	doc := domain.NewRecord(domain.Field{Name: "id", Value: "doc-1"}, domain.Field{Name: "title", Value: "Plan"})
	return row, domain.NewList([]domain.Record{doc}).WithPage(page)
}

// TestTextOfAFullPageNamesTheNextOffset checks that a list page with more rows to come names the offset of the next page,
// and that a last page names none.
func TestTextOfAFullPageNamesTheNextOffset(t *testing.T) {
	row, full := pagedDocuments(t, domain.NewPage(0, 1, 1))
	var b bytes.Buffer
	writeText(&b, row, full)
	if !strings.Contains(b.String(), "More rows may follow. Next page: --offset 1") {
		t.Fatalf("text of a full page = %q, want the next offset", b.String())
	}

	row, last := pagedDocuments(t, domain.NewPage(0, 5, 1))
	b.Reset()
	writeText(&b, row, last)
	if strings.Contains(b.String(), "--offset") {
		t.Fatalf("text of a last page = %q, want no next offset", b.String())
	}
}

// TestJSONOfAPagedListCarriesHasMoreAndTheNextOffset checks the JSON document of a paged list: has_more is always there,
// and next_offset only when more rows may follow.
func TestJSONOfAPagedListCarriesHasMoreAndTheNextOffset(t *testing.T) {
	row, full := pagedDocuments(t, domain.NewPage(10, 1, 1))
	doc := jsonDocument(row, full)
	if doc["has_more"] != true || doc["next_offset"] != 11 {
		t.Fatalf("JSON of a full page = %v, want has_more true and next_offset 11", doc)
	}

	row, last := pagedDocuments(t, domain.NewPage(10, 5, 1))
	doc = jsonDocument(row, last)
	if doc["has_more"] != false {
		t.Fatalf("JSON of a last page = %v, want has_more false", doc)
	}
	if _, ok := doc["next_offset"]; ok {
		t.Fatalf("JSON of a last page = %v, want no next_offset", doc)
	}
}

// TestUnpagedListsKeepTheirOldText checks that a list with no paging prints no continuation and no paging keys.
func TestUnpagedListsKeepTheirOldText(t *testing.T) {
	row, ok := operations.Lookup(domain.OpDocumentsList)
	if !ok {
		t.Fatal("no documents.list row")
	}
	doc := domain.NewRecord(domain.Field{Name: "id", Value: "doc-1"})
	rec := domain.NewList([]domain.Record{doc})
	var b bytes.Buffer
	writeText(&b, row, rec)
	if strings.Contains(b.String(), "More rows") {
		t.Fatalf("an unpaged list printed a continuation: %q", b.String())
	}
	if _, ok := jsonDocument(row, rec)["has_more"]; ok {
		t.Fatal("an unpaged list carries has_more")
	}
}
