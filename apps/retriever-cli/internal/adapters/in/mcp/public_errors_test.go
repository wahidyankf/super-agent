package mcp

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestEveryPublicErrorReachesTheModelAsItsFixedText is the drift guard of the safe-error list. Every sentinel of
// domain.PublicErrors(), wrapped with a raw secret, must reach the model as its own fixed text, and the secret must
// reach neither the answer nor the log line. A sentinel added to the list is covered here without a new test.
func TestEveryPublicErrorReachesTheModelAsItsFixedText(t *testing.T) {
	for _, sentinel := range domain.PublicErrors() {
		t.Run(sentinel.Error(), func(t *testing.T) {
			wrapped := fmt.Errorf("%s: %w", sentinelKey, sentinel)
			var logs bytes.Buffer
			cs := connect(t, serviceWith(t, &fakeGateway{err: wrapped}, mustSource(t, "outline")), &logs)
			text, isErr := call(t, cs, "list_collections", nil)
			if !isErr || text != sentinel.Error() {
				t.Fatalf("answer is (%v, %q), want the fixed text %q", isErr, text, sentinel.Error())
			}
			assertNoSentinel(t, text, logs.String())
		})
	}
}
