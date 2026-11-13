package graphql

import (
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestInvalidIdentityIsPublicWithItsVariableNameOnly checks the GraphQL text of a malformed e-mail variable (ADR 010
// section 1.2): the variable name, never the value, directly or wrapped.
func TestInvalidIdentityIsPublicWithItsVariableNameOnly(t *testing.T) {
	const want = "EXAMPLE_JIRA_EMAIL is not a valid e-mail address"
	err := domain.InvalidIdentityError{Variable: "EXAMPLE_JIRA_EMAIL"}
	for _, e := range []error{err, fmt.Errorf("load: %w", err)} {
		got, ok := publicMessage(e)
		if !ok || got != want {
			t.Fatalf("publicMessage(%v) = %q, %v; want %q, true", e, got, ok, want)
		}
	}
}
