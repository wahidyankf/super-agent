package operations

import "testing"

// TestResourceOfAOneWordPathIsEmpty checks that a row whose command path has no second word has no resource, so it belongs
// to no group named by --tools.
func TestResourceOfAOneWordPathIsEmpty(t *testing.T) {
	if got := (Row{CLI: "outline"}).Resource(); got != "" {
		t.Fatalf("Resource of a one-word path = %q, want the empty string", got)
	}
}
