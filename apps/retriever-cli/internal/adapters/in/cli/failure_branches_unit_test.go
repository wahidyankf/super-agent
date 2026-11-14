package cli

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestTypedFailuresKeepTheirTextAndTheirCode checks each typed failure that a command can return from a flag or a source
// lookup: its printed text is the typed text, and its JSON code names the reason.
func TestTypedFailuresKeepTheirTextAndTheirCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		text string
		code string
	}{
		{
			name: "usage",
			err:  usageError{msg: "the --port value is out of range"},
			text: "the --port value is out of range",
			code: "retriever.outline.usage_invalid_argument",
		},
		{
			name: "catalog",
			err:  domain.CatalogError{Message: "the source catalog is invalid"},
			text: "the source catalog is invalid",
			code: "retriever.outline.catalog_invalid",
		},
		{
			name: "unknown source",
			err:  domain.UnknownSourceError{ID: "example"},
			text: `unknown source "example"; run retriever sources list`,
			code: "retriever.outline.unknown_source",
		},
		{
			name: "ambiguous source",
			err:  domain.AmbiguousSourceError{Kind: domain.KindOutline, IDs: []string{"a", "b"}},
			text: "more than one outline source exists (a, b); pass --source <id>",
			code: "retriever.outline.ambiguous_source",
		},
		{
			name: "access level",
			err:  domain.AccessLevelError{Source: "example", Have: domain.AccessRead, Need: domain.AccessWrite},
			text: domain.AccessLevelError{Source: "example", Have: domain.AccessRead, Need: domain.AccessWrite}.Error(),
			code: "retriever.outline.access_level_too_low",
		},
	}
	for _, c := range cases {
		if got := publicMessage(c.err); got != c.text {
			t.Errorf("%s: publicMessage = %q, want %q", c.name, got, c.text)
		}
		if got := errorCode(c.err, "outline"); got != c.code {
			t.Errorf("%s: errorCode = %q, want %q", c.name, got, c.code)
		}
	}
}

// TestAnErrorThatNamesAKindKeepsIt checks that withKind does not wrap an error that already names its kind, and wraps a
// plain error of another kind so its text and code carry that kind.
func TestAnErrorThatNamesAKindKeepsIt(t *testing.T) {
	named := domain.KindError{Kind: domain.KindKiwi, Err: errors.New("the case is missing")}
	if got := withKind(domain.KindJira, named); got != named {
		t.Fatalf("an error that names a kind must keep it, got %v", got)
	}
	plain := errors.New("the case is missing")
	wrapped := withKind(domain.KindKiwi, plain)
	var kindErr domain.KindError
	if !errors.As(wrapped, &kindErr) || kindErr.Kind != domain.KindKiwi {
		t.Fatalf("a plain error of kind kiwi must name the kind, got %v", wrapped)
	}
}
