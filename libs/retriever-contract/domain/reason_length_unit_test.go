package domain

import (
	"errors"
	"strings"
	"testing"
)

// TestValidateReasonRefusesAReasonLongerThanTheMaximum checks the length rule of a change's reason: a reason of exactly
// MaxReasonLength characters is accepted, and one character more is refused.
func TestValidateReasonRefusesAReasonLongerThanTheMaximum(t *testing.T) {
	if err := ValidateReason(strings.Repeat("a", MaxReasonLength)); err != nil {
		t.Fatalf("a reason of %d characters: %v", MaxReasonLength, err)
	}
	if err := ValidateReason(strings.Repeat("a", MaxReasonLength+1)); !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("a reason of %d characters: error = %v, want ErrInvalidReason", MaxReasonLength+1, err)
	}
}
