package mcp

import (
	"strings"
	"testing"
)

// TestFitRowsWithNoRowsKeepsNoneAndNamesNoTruncation checks the zero-row edge (finding m6): an empty list is not
// truncated, even when its answer is too large, because no row was left out.
func TestFitRowsWithNoRowsKeepsNoneAndNamesNoTruncation(t *testing.T) {
	small := func(kept []string, truncated bool) any {
		return map[string]any{"count": len(kept), "truncated": truncated}
	}
	kept, truncated, err := fitRows([]string{}, small)
	if err != nil || len(kept) != 0 || truncated {
		t.Fatalf("fitRows(empty) = %v, %v, %v; want no rows, not truncated, no error", kept, truncated, err)
	}

	huge := func([]string, bool) any {
		return map[string]any{"text": strings.Repeat("x", pageCap+1)}
	}
	kept, truncated, err = fitRows(nil, huge)
	if err != nil || len(kept) != 0 || truncated {
		t.Fatalf("fitRows(nil, too large) = %v, %v, %v; want no rows, not truncated, no error", kept, truncated, err)
	}
}

// TestFitRowsWhenNoRowFitsKeepsNoneAndNamesTruncation checks the other edge: rows exist, but even the empty answer is
// too large, so none is kept and the list is truncated.
func TestFitRowsWhenNoRowFitsKeepsNoneAndNamesTruncation(t *testing.T) {
	render := func(kept []string, truncated bool) any {
		return map[string]any{"pad": strings.Repeat("x", pageCap), "truncated": truncated}
	}
	kept, truncated, err := fitRows([]string{"a", "b"}, render)
	if err != nil || len(kept) != 0 || !truncated {
		t.Fatalf("fitRows = %v, %v, %v; want no rows, truncated, no error", kept, truncated, err)
	}
}
