package domain

import (
	"errors"
	"testing"
)

func TestCheckPageAcceptsTheRangeAndRefusesTheRest(t *testing.T) {
	cases := []struct {
		name   string
		offset int
		limit  int
		want   error
	}{
		{name: "the first page of the default size", offset: 0, limit: 25},
		{name: "the largest page", offset: 50, limit: 100},
		{name: "a negative offset", offset: -1, limit: 25, want: ErrInvalidOffset},
		{name: "a zero limit", offset: 0, limit: 0, want: ErrInvalidLimit},
		{name: "a limit above one hundred", offset: 0, limit: 101, want: ErrInvalidLimit},
		{name: "a negative limit", offset: 0, limit: -5, want: ErrInvalidLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckPage(tc.offset, tc.limit)
			if !errors.Is(got, tc.want) && !(got == nil && tc.want == nil) {
				t.Fatalf("CheckPage(%d, %d) = %v, want %v", tc.offset, tc.limit, got, tc.want)
			}
		})
	}
}

func TestNewSearchQueryRefusesBlankText(t *testing.T) {
	if _, err := NewSearchQuery(""); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("empty query error = %v, want ErrInvalidQuery", err)
	}
	if _, err := NewSearchQuery(" \t "); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("blank query error = %v, want ErrInvalidQuery", err)
	}
	got, err := NewSearchQuery(" setup ")
	if err != nil || got != " setup " {
		t.Fatalf("NewSearchQuery = %q, %v; want the text unchanged", got, err)
	}
}

func TestOperationUnavailableErrorNamesTheMethodAndMatchesTheSentinel(t *testing.T) {
	err := OperationUnavailableError{Method: "documents.search"}
	want := "this Outline server does not offer documents.search; its version may be older than v1.12.0"
	if err.Error() != want {
		t.Fatalf("Error = %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, ErrOperationUnavailable) {
		t.Fatal("the error does not match ErrOperationUnavailable")
	}
	if errors.Is(err, ErrDocumentNotFound) {
		t.Fatal("the error matches an unrelated sentinel")
	}
}
