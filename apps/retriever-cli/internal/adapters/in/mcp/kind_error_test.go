package mcp

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestEveryPublicErrorIsShownWithTheKindNameToTheModel checks that a Kiwi failure reaches the model with the kind's own
// name, and never with the word Outline.
func TestEveryPublicErrorIsShownWithTheKindNameToTheModel(t *testing.T) {
	for _, sentinel := range domain.PublicErrors() {
		err := domain.KindError{Kind: domain.KindKiwi, Err: sentinel}
		got, _ := publicError(err)
		if got == nil || got.Error() != err.Error() {
			t.Errorf("publicError(kiwi %q) = %v, want %q", sentinel, got, err.Error())
			continue
		}
		if strings.Contains(got.Error(), "Outline") {
			t.Errorf("publicError(kiwi %q) = %q names Outline", sentinel, got)
		}
	}
}

// TestNoSourceOfKindNamesItsKindToTheModel checks that the no-source text names the kind of the error.
func TestNoSourceOfKindNamesItsKindToTheModel(t *testing.T) {
	got, _ := publicError(domain.NoSourceOfKindError{Kind: domain.KindKiwi})
	if got == nil || got.Error() != "no kiwi source is configured" {
		t.Fatalf("publicError = %v, want %q", got, "no kiwi source is configured")
	}
}
