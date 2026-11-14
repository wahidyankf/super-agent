package sources

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// TestWithPageNamesTheNextPageOfAPagedListOnly checks the page of a read: a paged list gets the continuation of its page,
// from the offset and the limit of the call, and a list without those parameters or a reply that is not a list keeps no
// page.
func TestWithPageNamesTheNextPageOfAPagedListOnly(t *testing.T) {
	documents, ok := operations.Lookup(domain.OpDocumentsList)
	if !ok {
		t.Fatal("no documents.list row")
	}
	collections, ok := operations.Lookup(domain.OpCollectionsList)
	if !ok {
		t.Fatal("no collections.list row")
	}
	page := domain.NewList([]domain.Record{domain.NewRecord(domain.Field{Name: "id", Value: "a"})})
	args := domain.NewArgs(map[string]any{"offset": 20, "limit": 1})

	got := withPage(documents, args, page)
	p, ok := got.Page()
	if !ok || p.Offset != 20 || !p.HasMore || p.NextOffset != 21 {
		t.Fatalf("paged list page = %+v, %v; want more rows at 21 after offset 20", p, ok)
	}

	if _, ok := withPage(collections, args, page).Page(); ok {
		t.Fatal("a list with no offset and limit parameter got a page")
	}
	object := domain.NewRecord(domain.Field{Name: "id", Value: "a"})
	if _, ok := withPage(documents, args, object).Page(); ok {
		t.Fatal("a reply that is not a list got a page")
	}
}
