package domain

import "testing"

// TestNewPageNamesTheNextOffsetAndWhetherMoreRowsMayFollow checks the continuation of a page: a full page may be followed
// by more rows, and a short page or a page with no limit is the last one.
func TestNewPageNamesTheNextOffsetAndWhetherMoreRowsMayFollow(t *testing.T) {
	cases := []struct {
		name                 string
		offset, limit, count int
		wantHasMore          bool
		wantNext             int
	}{
		{"full first page", 0, 10, 10, true, 10},
		{"full later page", 20, 10, 10, true, 30},
		{"short last page", 25, 10, 5, false, 30},
		{"empty page past the end", 40, 10, 0, false, 40},
		{"no limit is never continued", 0, 0, 3, false, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := NewPage(c.offset, c.limit, c.count)
			if !p.Paged || p.Offset != c.offset {
				t.Fatalf("page = %+v, want paged at offset %d", p, c.offset)
			}
			if p.HasMore != c.wantHasMore || p.NextOffset != c.wantNext {
				t.Fatalf("page = %+v, want has_more %v and next offset %d", p, c.wantHasMore, c.wantNext)
			}
		})
	}
}

// TestRecordCarriesItsPageOnlyWhenOneIsSet checks that a record holds a page once it is set, and that a record without
// one reports no page.
func TestRecordCarriesItsPageOnlyWhenOneIsSet(t *testing.T) {
	list := NewList([]Record{NewRecord(Field{Name: "id", Value: "a"})})
	if _, ok := list.Page(); ok {
		t.Fatal("a list with no page reports one")
	}
	paged := list.WithPage(NewPage(0, 1, 1))
	page, ok := paged.Page()
	if !ok || !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("Page() = %+v, %v, want more rows at offset 1", page, ok)
	}
	if _, ok := list.Page(); ok {
		t.Fatal("WithPage changed the record it was called on")
	}
	if len(paged.Items()) != 1 {
		t.Fatalf("WithPage lost the items: %v", paged.Items())
	}
}
