package envfile

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// outlineVariable is the variable that holds the Outline key in the test fixtures.
const outlineVariable = "EXAMPLE_OUTLINE_API_KEY"

// outlineRef returns the credential reference of the Outline key, failing the test when it is invalid.
func outlineRef(t *testing.T) domain.CredentialRef {
	t.Helper()
	ref, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	return ref
}
