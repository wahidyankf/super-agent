package mcp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestInvalidIdentityGivesAFixedTextForTheModel checks that a malformed e-mail variable reaches a model with a fixed text
// that names no variable, because the name of a credential or an identity is never shown to a model.
func TestInvalidIdentityGivesAFixedTextForTheModel(t *testing.T) {
	const want = "the e-mail address for this source is not valid"
	err := domain.InvalidIdentityError{Variable: "EXAMPLE_JIRA_EMAIL"}
	for _, e := range []error{err, fmt.Errorf("load: %w", err)} {
		public, class := publicError(e)
		if public == nil || public.Error() != want {
			t.Fatalf("publicError(%v) = %v, want %q", e, public, want)
		}
		if class != "invalid identity" {
			t.Fatalf("outcome class = %q, want invalid identity", class)
		}
		if strings.Contains(public.Error(), "EXAMPLE_JIRA_EMAIL") {
			t.Fatal("the public error names the variable")
		}
	}
}
