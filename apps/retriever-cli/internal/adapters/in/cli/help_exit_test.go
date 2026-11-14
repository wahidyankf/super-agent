package cli

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// TestEveryNoMatchCommandDocumentsExitStatusOne checks the help page of every catalog read that can match nothing (its
// row sets NoMatch). Its exit status 1 is the empty result, so the page must name it, and the page is derived from the
// catalog rather than from one hand-written path (finding m8).
func TestEveryNoMatchCommandDocumentsExitStatusOne(t *testing.T) {
	checked := 0
	for _, row := range operations.Rows() {
		if !row.NoMatch {
			continue
		}
		checked++
		argv := append(append([]string{}, row.Path()...), "--help")
		code, out, _ := runRetriever(t, argv, outlineSetup{})
		if code != 0 {
			t.Fatalf("%s --help exit = %d, want 0", strings.Join(row.Path(), " "), code)
		}
		if !strings.Contains(out, "\n  1    the search matched no item") {
			t.Fatalf("%s --help does not document exit status 1 for an empty result:\n%s", strings.Join(row.Path(), " "), out)
		}
	}
	if checked == 0 {
		t.Fatal("no catalog read sets NoMatch, so the test checks nothing")
	}
}
