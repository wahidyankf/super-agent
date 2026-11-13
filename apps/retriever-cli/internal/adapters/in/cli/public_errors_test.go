package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// driftSentinel stands in for a raw secret that an upstream error could carry. No terminal text may hold it.
const driftSentinel = "sentinel-key-drift-guard"

// TestEveryPublicErrorShowsItsFixedTextInTheCLI is the drift guard of the safe-error list for the terminal. Every
// sentinel of domain.PublicErrors(), wrapped with a raw secret, must print as its own fixed text.
func TestEveryPublicErrorShowsItsFixedTextInTheCLI(t *testing.T) {
	for _, sentinel := range domain.PublicErrors() {
		t.Run(sentinel.Error(), func(t *testing.T) {
			got := publicMessage(fmt.Errorf("%s: %w", driftSentinel, sentinel))
			if got != sentinel.Error() {
				t.Fatalf("publicMessage = %q, want the fixed text %q", got, sentinel.Error())
			}
			if strings.Contains(got, driftSentinel) {
				t.Fatalf("publicMessage %q carries the raw secret", got)
			}
		})
	}
}
