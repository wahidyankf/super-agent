package cli

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// genericCode is the code of an error that no entry names. A public error must never fall back to it (finding M1).
const genericCode = "retriever.outline.failed"

// TestEveryPublicErrorHasItsOwnCode checks that each public error has a JSON error code of its own, and that none of
// them falls back to the generic code.
func TestEveryPublicErrorHasItsOwnCode(t *testing.T) {
	seen := map[string]error{}
	for _, err := range domain.PublicErrors() {
		code := errorCode(err, "outline")
		if code == genericCode {
			t.Errorf("%q has the generic code %s; add it to errorReasons", err, genericCode)
		}
		if other, dup := seen[code]; dup {
			t.Errorf("%q and %q share the code %s", err, other, code)
		}
		seen[code] = err
	}
}

// TestEveryPublicErrorIsShownWithItsOwnText checks that a public error reaches the terminal with its fixed text, and
// never with the generic text.
func TestEveryPublicErrorIsShownWithItsOwnText(t *testing.T) {
	for _, err := range domain.PublicErrors() {
		if got := publicMessage(err); got != err.Error() {
			t.Errorf("publicMessage(%q) = %q, want the error's own text", err, got)
		}
	}
}

// TestEveryReasonNamesAPublicError checks that the reason table holds no error the domain does not list as public,
// so a table entry can never be shown while the public list hides it.
func TestEveryReasonNamesAPublicError(t *testing.T) {
	public := map[error]bool{}
	for _, err := range domain.PublicErrors() {
		public[err] = true
	}
	for _, e := range errorReasons {
		if e.err == errNotConfigured {
			continue
		}
		if !public[e.err] {
			t.Errorf("errorReasons names %q, which domain.PublicErrors does not list", e.err)
		}
	}
}
