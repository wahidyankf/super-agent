package cli

import (
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// TestUsageShowsRequiredFlagsWithoutBrackets checks the usage line of every generated command of every kind: the first
// required parameter is the positional argument, and each other required parameter is a flag that the command cannot run
// without, so the line shows it bare. An optional flag stays in [brackets].
func TestUsageShowsRequiredFlagsWithoutBrackets(t *testing.T) {
	for _, kind := range []domain.SourceKind{domain.KindOutline, domain.KindKiwi, domain.KindJira} {
		for _, row := range operations.RowsOf(kind) {
			usage := readLeaf(row, row.CLI).usage
			positional := row.Positional()
			for _, p := range row.Params {
				if p.Name == positional {
					continue
				}
				bare := " --" + kebab(p.Name) + " " + metaOf(p)
				bracketed := " [--" + kebab(p.Name) + " " + metaOf(p) + "]"
				if p.Required && (!strings.Contains(usage, bare) || strings.Contains(usage, bracketed)) {
					t.Errorf("%s: usage %q shows required --%s in brackets, want it bare", row.ID, usage, kebab(p.Name))
				}
				if !p.Required && !strings.Contains(usage, bracketed) {
					t.Errorf("%s: usage %q lacks optional --%s in brackets", row.ID, usage, kebab(p.Name))
				}
			}
		}
	}
}

// TestKiwiWriteUsageNamesTheRequiredFlags pins the two usage lines that showed their required flags as optional.
func TestKiwiWriteUsageNamesTheRequiredFlags(t *testing.T) {
	tree := commandTree()
	want := map[string]string{
		"test-cases create": "retriever kiwi test-cases create <summary> --product <id> --category <value> --priority <value> " +
			"[--status <value>] [--text <value>] [--dry-run] [--source <id>]",
		"plans add-case": "retriever kiwi plans add-case <planId> --case-id <id> [--dry-run] [--source <id>]",
	}
	for path, usage := range want {
		c := findChild(tree, append([]string{"kiwi"}, strings.Fields(path)...)...)
		if c == nil || c.leaf == nil {
			t.Fatalf("retriever kiwi %s is missing from the command tree", path)
		}
		if c.leaf.usage != usage {
			t.Errorf("retriever kiwi %s usage = %q, want %q", path, c.leaf.usage, usage)
		}
	}
}

// TestAMissingRequiredFlagIsNamed checks that a leaf whose row has a required flag refuses a command line without it,
// and that the refusal names the flag, as a missing positional argument is named.
func TestAMissingRequiredFlagIsNamed(t *testing.T) {
	tree := commandTree()
	c := findChild(tree, "kiwi", "test-cases", "create")
	_, err := c.leaf.parse([]string{"Summary", "--product", "product-1"})
	if err == nil || !strings.Contains(err.Error(), "--category") || !strings.Contains(err.Error(), "--priority") ||
		strings.Contains(err.Error(), "--product") {
		t.Fatalf("parse error = %v, want it to name the missing --category and --priority only", err)
	}
	if _, err := c.leaf.parse([]string{"Summary", "--product", "product-1", "--category", "A", "--priority", "B"}); err != nil {
		t.Fatalf("parse with every required flag: %v", err)
	}
	add := findChild(tree, "kiwi", "plans", "add-case")
	if _, err := add.leaf.parse([]string{"plan-2"}); err == nil || !strings.Contains(err.Error(), "--case-id") {
		t.Fatalf("parse error = %v, want it to name the missing --case-id", err)
	}
}
