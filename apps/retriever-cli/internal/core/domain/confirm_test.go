package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestConfirmErrorMatchesItsKindOfRefusal checks that a missing --confirm matches only ErrConfirmRequired, and a wrong
// --confirm matches only ErrConfirmMismatch.
func TestConfirmErrorMatchesItsKindOfRefusal(t *testing.T) {
	required := ConfirmError{Kind: "the document ID"}
	if !errors.Is(required, ErrConfirmRequired) || errors.Is(required, ErrConfirmMismatch) {
		t.Fatal("a missing --confirm must match ErrConfirmRequired and not ErrConfirmMismatch")
	}
	mismatch := ConfirmError{Kind: "the document ID", Mismatch: true}
	if !errors.Is(mismatch, ErrConfirmMismatch) || errors.Is(mismatch, ErrConfirmRequired) {
		t.Fatal("a wrong --confirm must match ErrConfirmMismatch and not ErrConfirmRequired")
	}
	if !strings.Contains(required.Error(), "the document ID") || !strings.Contains(mismatch.Error(), "the document ID") {
		t.Fatal("the refusals must name the kind of value to give")
	}
}

// TestValidateReasonBoundsLengthAndControlCharacters checks the reason rule: an empty reason is allowed, a reason of
// MaxReasonLength characters is allowed, and a longer reason or one with a control character is refused.
func TestValidateReasonBoundsLengthAndControlCharacters(t *testing.T) {
	if err := ValidateReason(""); err != nil {
		t.Fatalf("an empty reason = %v, want nil", err)
	}
	if err := ValidateReason(strings.Repeat("a", MaxReasonLength)); err != nil {
		t.Fatalf("a reason of %d characters = %v, want nil", MaxReasonLength, err)
	}
	if err := ValidateReason(strings.Repeat("a", MaxReasonLength+1)); !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("a reason of %d characters = %v, want ErrInvalidReason", MaxReasonLength+1, err)
	}
	if err := ValidateReason("a\x07b"); !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("a reason with a control character = %v, want ErrInvalidReason", err)
	}
}
