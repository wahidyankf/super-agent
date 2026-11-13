package cli

import (
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestInvalidIdentityIsShownWithItsVariableNameOnly checks the terminal text of a malformed e-mail variable (ADR 010
// section 1.2): the variable name, and never the value, directly or wrapped.
func TestInvalidIdentityIsShownWithItsVariableNameOnly(t *testing.T) {
	const want = "EXAMPLE_JIRA_EMAIL is not a valid e-mail address"
	err := domain.InvalidIdentityError{Variable: "EXAMPLE_JIRA_EMAIL"}
	if got := publicMessage(err); got != want {
		t.Fatalf("publicMessage = %q, want %q", got, want)
	}
	if got := publicMessage(fmt.Errorf("load: %w", err)); got != want {
		t.Fatalf("publicMessage(wrapped) = %q, want %q", got, want)
	}
}

// TestInvalidIdentityHasItsOwnErrorCode checks that a malformed e-mail variable has a JSON error code of its own, distinct
// from the missing-key code.
func TestInvalidIdentityHasItsOwnErrorCode(t *testing.T) {
	if got := errorCode(domain.InvalidIdentityError{Variable: "EXAMPLE_JIRA_EMAIL"}, "outline"); got != "retriever.outline.identity_invalid" {
		t.Fatalf("errorCode = %q, want retriever.outline.identity_invalid", got)
	}
}

// TestAccessAboveKindCeilingHasItsReasonCode checks the public ceiling error has its own reason code (ADR 010 section 1.1).
func TestAccessAboveKindCeilingHasItsReasonCode(t *testing.T) {
	if got := errorCode(domain.ErrAccessAboveKindCeiling, "outline"); got != "retriever.outline.access_above_kind_ceiling" {
		t.Fatalf("errorCode = %q, want retriever.outline.access_above_kind_ceiling", got)
	}
}
