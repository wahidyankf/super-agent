package mcp

import "testing"

// TestFitPlainRowsFailsLoudlyOnARowThatCannotEncode checks the broken-invariant path: a row of plain strings always
// encodes, so a row that cannot encode panics, and the handler's guard reports it as the unexpected error.
func TestFitPlainRowsFailsLoudlyOnARowThatCannotEncode(t *testing.T) {
	type row struct{ Bad chan int }
	defer func() {
		if recover() == nil {
			t.Fatal("a row that cannot encode must panic")
		}
	}()
	fitPlainRows([]row{{Bad: make(chan int)}}, func(kept []row, tr bool) any { return kept })
}
