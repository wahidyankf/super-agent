package graphql

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestEveryPublicErrorIsShownWithTheKindNameInGraphQL checks that a Kiwi failure reaches a GraphQL client with the
// kind's own name, and never with the word Outline.
func TestEveryPublicErrorIsShownWithTheKindNameInGraphQL(t *testing.T) {
	for _, sentinel := range domain.PublicErrors() {
		err := domain.KindError{Kind: domain.KindKiwi, Err: sentinel}
		got, ok := publicMessage(err)
		if !ok {
			t.Fatalf("publicMessage(kiwi %q) is not public", sentinel)
		}
		if got != err.Error() {
			t.Errorf("publicMessage(kiwi %q) = %q, want %q", sentinel, got, err.Error())
		}
		if strings.Contains(got, "Outline") {
			t.Errorf("publicMessage(kiwi %q) = %q names Outline", sentinel, got)
		}
	}
}

// TestKindErrorKeepsTheKeyRejectedTextInGraphQL checks the Kiwi key text for the GraphQL presenter.
func TestKindErrorKeepsTheKeyRejectedTextInGraphQL(t *testing.T) {
	got, ok := publicMessage(domain.KindError{Kind: domain.KindKiwi, Err: domain.ErrKeyRejected})
	if !ok || got != "Kiwi rejected the API key" {
		t.Fatalf("publicMessage = (%q, %v), want (%q, true)", got, ok, "Kiwi rejected the API key")
	}
}

// TestNoSourceOfKindNamesItsKindInGraphQL checks that the no-source text names the kind of the error.
func TestNoSourceOfKindNamesItsKindInGraphQL(t *testing.T) {
	got, ok := publicMessage(domain.NoSourceOfKindError{Kind: domain.KindKiwi})
	if !ok || got != "no kiwi source is configured" {
		t.Fatalf("publicMessage = (%q, %v), want (%q, true)", got, ok, "no kiwi source is configured")
	}
}
