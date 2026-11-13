package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestKindErrorCodesCarryTheKindName checks that the JSON error code of a non-Outline kind names that kind, and that the
// rest of the code is the code the Outline error has.
func TestKindErrorCodesCarryTheKindName(t *testing.T) {
	for _, e := range errorReasons {
		if e.err == errNotConfigured {
			continue
		}
		outline := errorCode(e.err, "outline")
		kiwi := errorCode(domain.KindError{Kind: domain.KindKiwi, Err: e.err}, "outline")
		want := strings.Replace(outline, "retriever.outline.", "retriever.kiwi.", 1)
		if kiwi != want {
			t.Errorf("errorCode(KindError kiwi %q) = %q, want %q", e.err, kiwi, want)
		}
	}
}

// TestKindErrorCodeForAMissingKeyNamesTheKind checks that a missing key on a kiwi source carries the kiwi prefix.
func TestKindErrorCodeForAMissingKeyNamesTheKind(t *testing.T) {
	err := domain.KindError{Kind: domain.KindKiwi, Err: domain.MissingKeyError{Variable: "EXAMPLE_KIWI_API_KEY"}}
	if got := errorCode(err, "outline"); got != "retriever.kiwi.key_missing" {
		t.Fatalf("errorCode = %q, want retriever.kiwi.key_missing", got)
	}
	if got := errorCode(domain.MissingKeyError{Variable: "EXAMPLE_OUTLINE_API_KEY"}, "outline"); got != "retriever.outline.key_missing" {
		t.Fatalf("Outline errorCode = %q, want retriever.outline.key_missing", got)
	}
}

// TestEveryPublicErrorIsShownWithTheKindNameInTheCLI checks that a non-Outline kind prints the sentinel text with its
// own name, and never the word Outline.
func TestEveryPublicErrorIsShownWithTheKindNameInTheCLI(t *testing.T) {
	for _, sentinel := range domain.PublicErrors() {
		err := domain.KindError{Kind: domain.KindKiwi, Err: sentinel}
		got := publicMessage(err)
		if got != err.Error() {
			t.Errorf("publicMessage(kiwi %q) = %q, want %q", sentinel, got, err.Error())
		}
		if strings.Contains(got, "Outline") {
			t.Errorf("publicMessage(kiwi %q) = %q names Outline", sentinel, got)
		}
	}
}

// TestKindErrorKeepsTheKeyRejectedText checks the owner's case: a rejected Kiwi key is reported for Kiwi.
func TestKindErrorKeepsTheKeyRejectedText(t *testing.T) {
	err := domain.KindError{Kind: domain.KindKiwi, Err: domain.ErrKeyRejected}
	if got := publicMessage(err); got != "Kiwi rejected the API key" {
		t.Fatalf("publicMessage = %q, want %q", got, "Kiwi rejected the API key")
	}
}

// TestNoSourceOfKindNamesItsKind checks that the no-source text comes from the error, so a kiwi command never says
// "no outline source".
func TestNoSourceOfKindNamesItsKind(t *testing.T) {
	err := domain.NoSourceOfKindError{Kind: domain.KindKiwi}
	if got := publicMessage(err); got != "no kiwi source is configured" {
		t.Fatalf("publicMessage = %q, want %q", got, "no kiwi source is configured")
	}
	if !errors.Is(err, domain.ErrNoSourceOfKind) {
		t.Fatalf("NoSourceOfKindError does not match ErrNoSourceOfKind")
	}
	outline := domain.NoSourceOfKindError{Kind: domain.KindOutline}
	if got := publicMessage(outline); got != "no outline source is configured" {
		t.Fatalf("Outline publicMessage = %q, want %q", got, "no outline source is configured")
	}
}
