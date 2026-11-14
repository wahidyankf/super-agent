package connectorspec

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
)

// TestAnErrorWithNoPathNamesTheWholeDocument checks the text of a whole-document finding.
func TestAnErrorWithNoPathNamesTheWholeDocument(t *testing.T) {
	err := &Error{File: "x.yml", Rule: "is not a spec"}
	if got, want := err.Error(), "connector spec file x.yml: (document): is not a spec"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

// TestAParameterDefaultAndBoundsReachTheRow checks that a spec parameter's default, minimum, and maximum reach the row.
func TestAParameterDefaultAndBoundsReachTheRow(t *testing.T) {
	data := strings.Replace(wikiHeader+docsGet, "      - name: id\n        type: ID\n        in: path\n        required: true\n",
		"      - name: id\n        type: ID\n        in: path\n        required: true\n"+
			"      - name: limit\n        type: Int\n        in: query\n        required: false\n        default: 20\n"+
			"        min: 1\n        max: 50\n", 1)
	set := mustLoad(t, specFile("example-wiki.yml", data))
	row, _ := set.Operations.Lookup("example-wiki.docs.get")
	limit, ok := row.Param("limit")
	if !ok {
		t.Fatal("the limit parameter is missing from the row")
	}
	if limit.Default != 20 || limit.Min != 1 || limit.Max != 50 || limit.Required {
		t.Fatalf("limit = %+v, want default 20, bounds 1 to 50, optional", limit)
	}
}

// TestAKindNameLongerThanTheSpecAcceptsIsRefusedByTheSpecModule checks that a long kind name stops at the spec rule.
func TestAKindNameLongerThanTheSpecAcceptsIsRefusedByTheSpecModule(t *testing.T) {
	long := strings.Repeat("a", 33)
	data := strings.ReplaceAll(strings.Replace(wikiHeader+docsGet, "kind: example-wiki", "kind: "+long, 1), "example-wiki.", long+".")
	mustFail(t, "connector spec file bad.yml: /kind: is longer than this field allows", specFile("bad.yml", data))
}

// mustPanic runs f and requires it to panic, which is how a broken invariant fails.
func mustPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("no panic, want the invariant to fail loudly")
		}
	}()
	f()
}

// TestKindSetFailsLoudlyOnARepeatedKind checks the kind set the start-up builds, which never repeats a kind on input.
func TestKindSetFailsLoudlyOnARepeatedKind(t *testing.T) {
	mustPanic(t, func() { kindSet([]domain.KindDefinition{{Name: "outline"}}) })
}

// TestBuildCatalogFailsLoudlyOnTwoRowsWithOneOperationID checks the catalog build, which never repeats an ID on input.
func TestBuildCatalogFailsLoudlyOnTwoRowsWithOneOperationID(t *testing.T) {
	kinds := kindSet([]domain.KindDefinition{{Name: "example-wiki"}})
	row := operations.Row{ID: "example-wiki.docs.get", CLI: "example-wiki docs get"}
	mustPanic(t, func() { buildCatalog(kinds, []operations.Row{row, row}) })
}

// TestPlainValueReadsANumberAsAGoValue checks each branch of the default conversion.
func TestPlainValueReadsANumberAsAGoValue(t *testing.T) {
	if got := plainValue(json.Number("20")); got != 20 {
		t.Fatalf("plainValue(20) = %v, want int 20", got)
	}
	if got := plainValue(json.Number("1.5")); got != 1.5 {
		t.Fatalf("plainValue(1.5) = %v, want 1.5", got)
	}
	if got := plainValue(json.Number("x")); got != json.Number("x") {
		t.Fatalf("plainValue(x) = %v, want the number as it was", got)
	}
	if got := plainValue("text"); got != "text" {
		t.Fatalf("plainValue(text) = %v, want text", got)
	}
}
