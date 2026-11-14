package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestPublicMessageMapsSourceSelectionErrors checks the terminal text of each source-selection refusal, bare and
// wrapped, so no raw detail is printed.
func TestPublicMessageMapsSourceSelectionErrors(t *testing.T) {
	ambiguous := domain.AmbiguousSourceError{Kind: domain.KindOutline, IDs: []string{"outline", "outline-engineering"}}
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "an unknown source", err: domain.UnknownSourceError{ID: "rnd"}, want: `unknown source "rnd"; run retriever sources list`},
		{name: "an unknown source, wrapped", err: fmt.Errorf("resolve: %w", domain.UnknownSourceError{ID: "rnd"}), want: `unknown source "rnd"; run retriever sources list`},
		{name: "an ambiguous source", err: ambiguous, want: "more than one outline source exists (outline, outline-engineering); pass --source <id>"},
		{name: "an ambiguous source, wrapped", err: fmt.Errorf("resolve: %w", ambiguous), want: "more than one outline source exists (outline, outline-engineering); pass --source <id>"},
		{name: "no source of the kind", err: domain.ErrNoSourceOfKind, want: "no outline source is configured"},
		{name: "no source of the kind, wrapped", err: fmt.Errorf("resolve: %w", domain.ErrNoSourceOfKind), want: "no outline source is configured"},
		{name: "a source of another kind", err: domain.ErrWrongSourceKind, want: "the source is of another kind"},
		{name: "a source of another kind, wrapped", err: fmt.Errorf("resolve: %w", domain.ErrWrongSourceKind), want: "the source is of another kind"},
		{name: "an invalid source ID", err: domain.ErrInvalidSourceID, want: "invalid source ID"},
		{name: "an unknown error gets the generic text", err: errors.New("raw upstream detail"), want: "unexpected error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := publicMessage(tc.err); got != tc.want {
				t.Fatalf("publicMessage = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPublicMessageNamesOnlyTheMissingVariable checks that a missing key names its variable and no value.
func TestPublicMessageNamesOnlyTheMissingVariable(t *testing.T) {
	got := publicMessage(domain.MissingKeyError{Variable: "EXAMPLE_OUTLINE_API_KEY"})
	if got != "EXAMPLE_OUTLINE_API_KEY is not set" {
		t.Fatalf("publicMessage = %q", got)
	}
}
