package operations

import (
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestBindRefusesTheStandardInputMarkerAsText covers finding B1: the help of a text parameter once said that "-" read
// standard input, which the catalog never did, so "-" was sent as the literal text. Stdin is not implemented, so a text
// whose whole value is "-" is refused before any call. Only that exact value is refused.
func TestBindRefusesTheStandardInputMarkerAsText(t *testing.T) {
	row, ok := Lookup(domain.OpDocumentsCreate)
	if !ok {
		t.Fatal("documents.create is not in the catalog")
	}
	if _, err := Bind(row, map[string]any{"title": "Plan", "text": "-"}); !errors.Is(err, domain.ErrStandardInputUnsupported) {
		t.Fatalf("Bind with text \"-\" = %v, want ErrStandardInputUnsupported", err)
	}
	for _, text := range []string{"- ", " -", "--", "a-", "-a"} {
		if _, err := Bind(row, map[string]any{"title": "Plan", "text": text}); err != nil {
			t.Errorf("Bind with text %q = %v, want it accepted", text, err)
		}
	}
}

// TestEveryTextParameterRefusesTheStandardInputMarker checks the rule on every text parameter of the catalog, so a new
// text parameter cannot send "-" as its literal text.
func TestEveryTextParameterRefusesTheStandardInputMarker(t *testing.T) {
	found := 0
	for _, row := range Rows() {
		for _, p := range row.Params {
			if p.Kind != KindText {
				continue
			}
			found++
			if _, err := convert(p, "-"); !errors.Is(err, domain.ErrStandardInputUnsupported) {
				t.Errorf("%s %s: convert(\"-\") = %v, want ErrStandardInputUnsupported", row.ID, p.Name, err)
			}
		}
	}
	if found == 0 {
		t.Fatal("the catalog has no text parameter to check")
	}
}

// TestNoTextHelpPromisesStandardInput checks that no parameter help still promises to read standard input.
func TestNoTextHelpPromisesStandardInput(t *testing.T) {
	for _, row := range Rows() {
		for _, p := range row.Params {
			if p.Kind == KindText && containsStdinPromise(p.Help) {
				t.Errorf("%s %s: help %q promises standard input", row.ID, p.Name, p.Help)
			}
		}
	}
}

// containsStdinPromise reports whether a help text says that standard input is read.
func containsStdinPromise(help string) bool {
	for _, phrase := range []string{"standard input", "standard in", "from stdin", "read from -"} {
		if strings.Contains(help, phrase) {
			return true
		}
	}
	return false
}
