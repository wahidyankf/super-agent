package domain

import (
	"errors"
	"testing"
)

// TestPublicErrorsAreDistinctNonEmptySentinels checks the shared list: every entry has a fixed, non-empty text, no
// entry is listed twice, and a caller that changes its copy does not change the next one.
func TestPublicErrorsAreDistinctNonEmptySentinels(t *testing.T) {
	list := PublicErrors()
	if len(list) == 0 {
		t.Fatal("PublicErrors is empty")
	}
	seen := map[error]bool{}
	for _, e := range list {
		if e == nil || e.Error() == "" {
			t.Fatalf("PublicErrors holds an empty entry: %v", e)
		}
		if seen[e] {
			t.Fatalf("PublicErrors lists %q twice", e)
		}
		seen[e] = true
	}
	list[0] = errors.New("changed by the caller")
	if PublicErrors()[0] == list[0] {
		t.Fatal("PublicErrors returns a shared slice")
	}
}
