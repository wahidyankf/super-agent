package unit_test

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// sentinelOfText returns the library sentinel whose fixed text is text, for the texts the scenarios name.
func sentinelOfText(t *testing.T, text string) error {
	t.Helper()
	sentinels := map[string]error{
		"not found":   domain.ErrNotFound,
		"invalid tag": domain.ErrInvalidTag,
		"the change may or may not have happened; check before you retry":  domain.ErrWriteOutcomeUnknown,
		"the catalog needs at least one source":                            domain.ErrEmptyCatalog,
		"invalid credential name":                                          domain.ErrInvalidCredentialRef,
		"invalid kind name: use a lowercase slug of at most 32 characters": domain.ErrInvalidKindName,
		"invalid plugin path: use an absolute path":                        domain.ErrInvalidPluginPath,
		"Outline rejected the API key":                                     domain.ErrKeyRejected,
	}
	sentinel, ok := sentinels[text]
	if !ok {
		t.Fatalf("the scenario names the error %q, which this binding does not map", text)
	}
	return sentinel
}

// @scenario specs/libs/retriever-contract/behaviours/public-errors.feature: An error on the public list keeps its fixed text
func TestPublicErrorsOnPublicListKeepFixedText(t *testing.T) {
	for _, text := range []string{
		"not found",
		"invalid tag",
		"the change may or may not have happened; check before you retry",
	} {
		t.Run(text, func(t *testing.T) {
			// The presenter shows a public error through a kind error, and the Outline kind shows the text unchanged.
			public, ok := domain.KindError{Kind: domain.KindOutline, Err: sentinelOfText(t, text)}.PublicText()
			if !ok {
				t.Fatalf("the error has no public text, want %q", text)
			}
			if public != text {
				t.Errorf("public text = %q, want %q", public, text)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/public-errors.feature: An error that is not on the public list has no public text
func TestPublicErrorsNotOnPublicListHaveNoPublicText(t *testing.T) {
	for _, text := range []string{
		"the catalog needs at least one source",
		"invalid credential name",
		"invalid kind name: use a lowercase slug of at most 32 characters",
		"invalid plugin path: use an absolute path",
	} {
		t.Run(text, func(t *testing.T) {
			public, ok := domain.KindError{Kind: domain.KindOutline, Err: sentinelOfText(t, text)}.PublicText()
			if ok {
				t.Errorf("public text = %q, want none", public)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/public-errors.feature: A kind error wrapping a public error shows its text with the kind name
func TestPublicErrorsKindErrorWrappingPublicErrorShowsTextWithKindName(t *testing.T) {
	kiwi, err := domain.BuiltinKinds().Kind("kiwi")
	noError(t, err)
	failure := domain.KindError{Kind: kiwi, Err: sentinelOfText(t, "Outline rejected the API key")}
	public, ok := failure.PublicText()
	if !ok {
		t.Fatalf("the failure has no public text, want one")
	}
	if want := "Kiwi rejected the API key"; public != want {
		t.Errorf("public text = %q, want %q", public, want)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/public-errors.feature: A kind error wrapping an error that is not public has no public text
func TestPublicErrorsKindErrorWrappingNonPublicErrorHasNoPublicText(t *testing.T) {
	kiwi, err := domain.BuiltinKinds().Kind("kiwi")
	noError(t, err)
	failure := domain.KindError{Kind: kiwi, Err: sentinelOfText(t, "the catalog needs at least one source")}
	if public, ok := failure.PublicText(); ok {
		t.Errorf("public text = %q, want none", public)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/public-errors.feature: A missing Outline operation names the operation and the version floor
func TestPublicErrorsMissingOutlineOperationNamesOperationAndVersionFloor(t *testing.T) {
	err := error(domain.OperationUnavailableError{Method: "documents.foo"})
	wantText(t, err, "this Outline server does not offer documents.foo; its version may be older than v1.12.0")
	if !errors.Is(err, domain.ErrOperationUnavailable) {
		t.Errorf("error does not match the operation unavailable error")
	}
}
