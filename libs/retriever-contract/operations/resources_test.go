package operations

import (
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestResourceIsTheWordAfterOutlineOfEveryRow checks that a row's resource is the second word of its command path, the
// word that names the group the row belongs to, such as documents for "outline documents get".
func TestResourceIsTheWordAfterOutlineOfEveryRow(t *testing.T) {
	if got := rowOf(t, domain.OpCollectionsDocuments).Resource(); got != "collections" {
		t.Fatalf("resource of collections.documents = %q, want collections", got)
	}
	if got := rowOf(t, domain.OpDocumentsGet).Resource(); got != "documents" {
		t.Fatalf("resource of documents.get = %q, want documents", got)
	}
	for _, row := range RowsOf(domain.KindOutline) {
		path := row.Path()
		if len(path) < 2 || path[0] != "outline" {
			t.Fatalf("row %s has the command %q, want outline <resource> ...", row.ID, row.CLI)
		}
		if row.Resource() != path[1] {
			t.Fatalf("row %s resource = %q, want %q", row.ID, row.Resource(), path[1])
		}
	}
}

// TestResourcesAreTheDistinctWordsOfTheCatalog checks that the resource words come from the catalog: every row's
// resource is one of them, each word is listed once, and the list is sorted. Nothing is written by hand.
func TestResourcesAreTheDistinctWordsOfTheCatalog(t *testing.T) {
	got := Resources()
	if !slices.IsSorted(got) {
		t.Fatalf("Resources() = %v, want sorted", got)
	}
	if len(slices.Compact(slices.Clone(got))) != len(got) {
		t.Fatalf("Resources() = %v holds a word twice", got)
	}
	for _, row := range RowsOf(domain.KindOutline) {
		if !slices.Contains(got, row.Resource()) {
			t.Fatalf("Resources() does not hold the resource %q of row %s", row.Resource(), row.ID)
		}
	}
	for _, want := range []string{"documents", "comments", "collections", "access-requests", "webhook-subscriptions"} {
		if !slices.Contains(got, want) {
			t.Fatalf("Resources() = %v, want it to hold %q", got, want)
		}
	}
}

// TestParseResourcesReadsANamedList checks the accepted forms: one resource, several in any order, and a repeated word,
// which is kept once in the order it first appears.
func TestParseResourcesReadsANamedList(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{raw: "documents", want: []string{"documents"}},
		{raw: "documents,comments", want: []string{"documents", "comments"}},
		{raw: "comments,documents,comments", want: []string{"comments", "documents"}},
		{raw: "access-requests,webhook-subscriptions", want: []string{"access-requests", "webhook-subscriptions"}},
	}
	for _, tc := range cases {
		got, err := ParseResources(tc.raw)
		if err != nil {
			t.Fatalf("ParseResources(%q): %v", tc.raw, err)
		}
		if !slices.Equal(got, tc.want) {
			t.Fatalf("ParseResources(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

// TestParseResourcesRefusesAnEmptyValueOrEntry checks that an empty value and an empty entry are both refused: a
// trailing, leading, or doubled comma names nothing, and the list must name something.
func TestParseResourcesRefusesAnEmptyValueOrEntry(t *testing.T) {
	for _, raw := range []string{"", ",", "documents,", ",documents", "documents,,comments"} {
		if _, err := ParseResources(raw); err == nil {
			t.Fatalf("ParseResources(%q) accepted an empty value or entry", raw)
		}
	}
}

// TestParseResourcesRefusesAnUnknownResourceAndListsTheValidOnes checks that an unknown word is refused, matching is exact
// (case and spaces count), and the error lists the valid resources, so the user can correct the flag.
func TestParseResourcesRefusesAnUnknownResourceAndListsTheValidOnes(t *testing.T) {
	for _, raw := range []string{"root", "Documents", " comments", "documents,root"} {
		_, err := ParseResources(raw)
		if err == nil {
			t.Fatalf("ParseResources(%q) accepted an unknown resource", raw)
		}
		if !strings.Contains(err.Error(), "unknown resource") || !strings.Contains(err.Error(), "webhook-subscriptions") {
			t.Fatalf("ParseResources(%q) error = %q, want the unknown word and the valid resources", raw, err)
		}
	}
}
