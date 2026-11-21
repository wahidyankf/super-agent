package unit_test

import (
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// @scenario specs/libs/retriever-contract/behaviours/records-and-pages.feature: A bound argument is read by name, and an unset one reads as zero
func TestRecordsReadBoundArgumentByNameAndUnsetOneReadsAsZero(t *testing.T) {
	args := domain.NewArgs(map[string]any{"limit": 25, "query": "install"})
	cases := []struct {
		name  string
		kind  string
		value any
	}{
		{"limit", "number", 25},
		{"query", "text", "install"},
		{"offset", "number", 0},
		{"missing", "text", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got any
			if tc.kind == "number" {
				got = args.Int(tc.name)
			} else {
				got = args.String(tc.name)
			}
			if got != tc.value {
				t.Errorf("argument %q as %s = %v, want %v", tc.name, tc.kind, got, tc.value)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/records-and-pages.feature: Changing the input map later does not change the arguments
func TestRecordsChangingInputMapLaterDoesNotChangeArguments(t *testing.T) {
	input := map[string]any{"query": "install"}
	args := domain.NewArgs(input)
	input["query"] = "other"
	if got := args.String("query"); got != "install" {
		t.Errorf("query = %q, want %q", got, "install")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/records-and-pages.feature: Changing a returned list does not change the arguments
func TestRecordsChangingReturnedListDoesNotChangeArguments(t *testing.T) {
	args := domain.NewArgs(map[string]any{"tags": []string{"a", "b"}})
	readBack := args.Strings("tags")
	readBack[0] = "x"
	if got, want := args.Strings("tags"), []string{"a", "b"}; !sameStrings(got, want) {
		t.Errorf("tags = %v, want %v", got, want)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/records-and-pages.feature: A page reports whether more rows follow and where the next one starts
func TestRecordsPageReportsWhetherMoreRowsFollowAndNextOffset(t *testing.T) {
	cases := []struct {
		offset, limit, count int
		more                 bool
		next                 int
	}{
		{0, 2, 2, true, 2},
		{0, 2, 1, false, 1},
		{0, 0, 5, false, 5},
		{10, 25, 25, true, 35},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("offset %d limit %d count %d", tc.offset, tc.limit, tc.count), func(t *testing.T) {
			record := domain.NewList(nil).WithPage(domain.NewPage(tc.offset, tc.limit, tc.count))
			page, ok := record.Page()
			if !ok {
				t.Fatalf("the record holds no page")
			}
			if page.HasMore != tc.more {
				t.Errorf("more rows may follow = %v, want %v", page.HasMore, tc.more)
			}
			if page.NextOffset != tc.next {
				t.Errorf("next offset = %d, want %d", page.NextOffset, tc.next)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/records-and-pages.feature: A list record keeps the page it was given
func TestRecordsListRecordKeepsPageItWasGiven(t *testing.T) {
	items := []domain.Record{
		domain.NewRecord(domain.Field{Name: "name", Value: "Engineering"}),
		domain.NewRecord(domain.Field{Name: "name", Value: "Support"}),
	}
	list := domain.NewList(items).WithPage(domain.NewPage(0, 2, 2))
	page, ok := list.Page()
	if !ok {
		t.Fatalf("the page is absent, want it present")
	}
	if page.Offset != 0 {
		t.Errorf("page offset = %d, want 0", page.Offset)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/records-and-pages.feature: A record that is not a list reports no page
func TestRecordsRecordThatIsNotListReportsNoPage(t *testing.T) {
	record := domain.NewRecord(domain.Field{Name: "name", Value: "Engineering"})
	if _, ok := record.Page(); ok {
		t.Errorf("a page is present, want none")
	}
}

// @scenario specs/libs/retriever-contract/behaviours/records-and-pages.feature: A record field is read by name, and a missing field is absent
func TestRecordsFieldIsReadByNameAndMissingFieldIsAbsent(t *testing.T) {
	record := domain.NewRecord(
		domain.Field{Name: "name", Value: "Engineering"},
		domain.Field{Name: "count", Value: 3},
	)
	cases := []struct {
		name   string
		want   any
		absent bool
	}{
		{"name", "Engineering", false},
		{"count", 3, false},
		{"owner", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, ok := record.Get(tc.name)
			if tc.absent {
				if ok {
					t.Errorf("field %q is present with %v, want absent", tc.name, value)
				}
				return
			}
			if !ok || value != tc.want {
				t.Errorf("field %q = %v (present %v), want %v", tc.name, value, ok, tc.want)
			}
		})
	}
}
