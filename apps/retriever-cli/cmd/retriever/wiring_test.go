package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// emptyEnv reads every variable as unset, so no test seam is on.
func emptyEnv(string) string { return "" }

// TestReadSeamsIsOffWhenNothingIsSet checks that an unset environment turns every test seam off, so each source uses its own
// endpoint.
func TestReadSeamsIsOffWhenNothingIsSet(t *testing.T) {
	seams, err := readSeams(emptyEnv)
	if err != nil {
		t.Fatalf("readSeams: %v", err)
	}
	for _, kind := range seamKinds {
		if seams[kind].on {
			t.Fatalf("the %s seam is on with an empty environment", kind)
		}
	}
	if err := checkSeams(seams); err != nil {
		t.Fatalf("checkSeams with every seam off: %v", err)
	}
}

// TestStartupFailureWritesOneLineAndExitsTwo checks that a failure before the CLI runs goes to standard error only, and
// that its status is 2, the status of every failure that is not a result (CLI convention 001).
func TestStartupFailureWritesOneLineAndExitsTwo(t *testing.T) {
	var stderr bytes.Buffer
	code := startupFailure(&stderr, []string{"sources", "list"}, os.ErrInvalid)
	if code != 2 || stderr.String() != "retriever: "+os.ErrInvalid.Error()+"\n" {
		t.Fatalf("startupFailure = %d with %q, want 2 and one line", code, stderr.String())
	}
}

// TestSourceKindsHaveAStableOrder checks the order the seams are read in, which the startup error reports first.
func TestSourceKindsHaveAStableOrder(t *testing.T) {
	want := []domain.SourceKind{domain.KindOutline, domain.KindKiwi, domain.KindJira}
	if len(seamKinds) != len(want) {
		t.Fatalf("seamKinds = %v, want %v", seamKinds, want)
	}
	for i := range want {
		if seamKinds[i] != want[i] {
			t.Fatalf("seamKinds = %v, want %v", seamKinds, want)
		}
	}
}
