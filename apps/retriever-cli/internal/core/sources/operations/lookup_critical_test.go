package operations

import (
	"testing"
)

// TestIsCriticalFollowsTheCriticalBlock checks that a row with no critical block is never critical, and that a critical
// block with no condition is always critical.
func TestIsCriticalFollowsTheCriticalBlock(t *testing.T) {
	if (Row{}).IsCritical(map[string]any{}) {
		t.Fatal("a row with no critical block is critical; want not critical")
	}
	always := Row{Critical: &Critical{}}
	if !always.IsCritical(nil) {
		t.Fatal("a critical block with no condition is not critical; want critical")
	}
}

// TestLookupReturnsFalseForAnUnknownOperation checks that an operation ID the catalog does not hold is not found.
func TestLookupReturnsFalseForAnUnknownOperation(t *testing.T) {
	if _, ok := Lookup("no.such.operation"); ok {
		t.Fatal("Lookup found an operation that the catalog does not hold")
	}
}

// TestResourceIsEmptyForACommandPathOfOneWord checks that a row whose command path has no resource word names none.
func TestResourceIsEmptyForACommandPathOfOneWord(t *testing.T) {
	if got := (Row{CLI: "outline"}).Resource(); got != "" {
		t.Fatalf("Resource of a one-word path = %q; want empty", got)
	}
}
