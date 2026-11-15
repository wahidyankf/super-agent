package architecture

import (
	"bytes"
	"sort"
	"strings"
	"testing"
)

// The rule is pure: it reads the sources it is given and touches no disk, so its negative tests run untagged. The
// repository walk that feeds it is in bundled_calls_integration_test.go.

// bundledCallMarker is the call that only the loader of the embedded specs may make: a qualified call, so the
// definition of the entry itself does not count.
var bundledCallMarker = []byte(".ParseBundled(")

// bundledCallers returns the sorted paths of the production Go files that call ParseBundled: every .go file that is
// not a test. A test may exercise the embedded entry, but only the embed loader may serve it.
func bundledCallers(files map[string][]byte) []string {
	var out []string
	for path, data := range files {
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		if bytes.Contains(data, bundledCallMarker) {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

// TestBundledCallersFindsTheProductionCallers checks the rule on in-memory sources: a production file that calls the
// embedded entry is listed, and tests and other files are not.
func TestBundledCallersFindsTheProductionCallers(t *testing.T) {
	files := map[string][]byte{
		"apps/retriever-cli/internal/connectorspec/bundled.go":      []byte("package connectorspec\nvar _ = spec.ParseBundled(nil)\n"),
		"apps/retriever-cli/internal/connectorspec/bundled_test.go": []byte("package connectorspec\nvar _ = spec.ParseBundled(nil)\n"),
		"libs/retriever-spec/bundled.go":                            []byte("package spec\nfunc ParseBundled() {}\n"),
		"libs/retriever-spec/validate.go":                           []byte("package spec\nfunc Parse() {}\n"),
		"docs/note.md":                                              []byte("ParseBundled(\n"),
	}
	got := bundledCallers(files)
	if len(got) != 1 || got[0] != "apps/retriever-cli/internal/connectorspec/bundled.go" {
		t.Fatalf("callers = %q, want the embed loader only", got)
	}
}

// TestBundledCallersFindsNoCaller checks the empty case: with no production caller, the rule reports none.
func TestBundledCallersFindsNoCaller(t *testing.T) {
	files := map[string][]byte{
		"libs/retriever-spec/bundled_test.go": []byte("package spec\nvar _ = ParseBundled(nil)\n"),
	}
	if got := bundledCallers(files); len(got) != 0 {
		t.Fatalf("callers = %q, want none", got)
	}
}
