package cli

import (
	"bytes"
	_ "embed"
	"slices"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/mcp"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/conformance"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The stored texts the suite compares the bundled specs against live beside
// this test. A deliberate change to a command, a tool, or the review updates
// the stored text in the same change, after reading the diff.

//go:embed testdata/conformance/outline.cli.golden
var outlineCLIGolden []byte

//go:embed testdata/conformance/outline.mcp.golden
var outlineMCPGolden []byte

//go:embed testdata/conformance/outline.review.golden
var outlineReviewGolden []byte

//go:embed testdata/conformance/jira.cli.golden
var jiraCLIGolden []byte

//go:embed testdata/conformance/jira.mcp.golden
var jiraMCPGolden []byte

//go:embed testdata/conformance/jira.review.golden
var jiraReviewGolden []byte

// bundledGoldens maps a bundled file name to its three stored texts: the CLI
// help, the MCP tools, and the review. A bundled spec with no entry fails the
// suite, so a new bundled spec lands with its stored texts.
var bundledGoldens = map[string]struct{ cli, mcp, review []byte }{
	"outline.yml": {cli: outlineCLIGolden, mcp: outlineMCPGolden, review: outlineReviewGolden},
	"jira.yml":    {cli: jiraCLIGolden, mcp: jiraMCPGolden, review: jiraReviewGolden},
}

// TestBundledSpecsPassConformance runs the suite over every bundled spec
// present: each model validates, names only example hosts, replays every
// example, claims every catalog row of its kind, and derives the stored CLI,
// MCP, and README texts byte for byte. The specs are enumerated, never
// counted, so the Kiwi scenarios staying Go-served changes nothing here.
func TestBundledSpecsPassConformance(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v, want nil", err)
	}
	if len(set.Specs) == 0 {
		t.Fatal("LoadBundled() holds no bundled spec")
	}
	for _, sp := range set.Specs {
		t.Run(sp.File, func(t *testing.T) {
			checkBundledSpec(t, sp)
		})
	}
}

// checkBundledSpec runs every check of the suite over one bundled spec.
func checkBundledSpec(t *testing.T, sp connectorspec.Spec) {
	t.Helper()
	if issues := conformance.CheckModel(sp.Model); len(issues) != 0 {
		t.Fatalf("CheckModel() issues = %v, want none", issues)
	}
	if issues := conformance.CheckHosts(sp.Model); len(issues) != 0 {
		t.Fatalf("CheckHosts() issues = %v, want none", issues)
	}
	issues, values := conformance.Test(sp.Model)
	if len(issues) != 0 {
		t.Fatalf("Test() issues = %v, want none", issues)
	}
	wantValues := 0
	for _, op := range sp.Model.Operations {
		wantValues += len(op.Examples)
	}
	if len(values) != wantValues {
		t.Fatalf("Test() replayed %d examples, want %d", len(values), wantValues)
	}
	checkCatalogClaimed(t, sp)
	checkDerivedTexts(t, sp)
}

// checkCatalogClaimed proves L10 for the kind of one bundled spec: every
// catalog row of the kind is claimed by exactly one operation of the spec,
// through its catalog_id.
func checkCatalogClaimed(t *testing.T, sp connectorspec.Spec) {
	t.Helper()
	kind := sp.Kind
	claimed := map[string]int{}
	for _, op := range sp.Model.Operations {
		if op.CatalogID != "" {
			claimed[op.CatalogID]++
		}
	}
	var missing []string
	for _, row := range operations.RowsOf(kind) {
		if n := claimed[string(row.ID)]; n != 1 {
			missing = append(missing, string(row.ID))
		}
	}
	if len(missing) != 0 {
		t.Fatalf("unclaimed or twice-claimed catalog rows of %s: %v", kind, missing)
	}
}

// checkDerivedTexts compares the CLI, MCP, and README texts the real adapters
// derive from one bundled spec with the stored texts.
func checkDerivedTexts(t *testing.T, sp connectorspec.Spec) {
	t.Helper()
	stored, ok := bundledGoldens[sp.File]
	if !ok {
		t.Fatalf("no stored texts for bundled spec %s", sp.File)
	}
	if got := bundledCLIText(sp.Kind); !equalText(got, stored.cli) {
		t.Error("the CLI text derived from the spec differs from the stored text")
	}
	tools := string(mcp.ToolsJSON(bundledToolRows(sp.Kind)))
	if !equalText(tools, stored.mcp) {
		t.Error("the MCP text derived from the spec differs from the stored text")
	}
	if got := spec.Review(sp.Model); !equalText(got, stored.review) {
		t.Error("the review derived from the spec differs from the stored text")
	}
}

// equalText reports whether a derived text equals its stored text, byte for
// byte.
func equalText(got string, stored []byte) bool {
	return got == string(stored)
}

// bundledCLIText returns the help of the command-line surface of one bundled
// kind: the help of its group, then the help of each of its commands in
// catalog order. The text comes from the real command tree.
func bundledCLIText(kind domain.SourceKind) string {
	tree := commandTree()
	group, path, _ := walk(tree, []string{kind.String()})
	var b bytes.Buffer
	writeHelp(&b, path, group)
	for _, row := range operations.RowsOf(kind) {
		words := strings.Split(row.CLI, " ")
		node, lpath, idx := walk(tree, words)
		if node.leaf == nil || idx != len(words) {
			continue
		}
		b.WriteString("\n")
		writeHelp(&b, lpath, node)
	}
	return b.String()
}

// bundledToolRows returns the catalog rows of one bundled kind that name an
// MCP tool, in catalog order.
func bundledToolRows(kind domain.SourceKind) []operations.Row {
	var rows []operations.Row
	for _, row := range operations.RowsOf(kind) {
		if row.Tool != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

// TestBundledHostCheckFiresOnAnOutsideHost proves gate 4 at the level of the
// suite: a bundled model whose suffix leaves the example domains fails the
// host check, so the suite would fail it.
func TestBundledHostCheckFiresOnAnOutsideHost(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v, want nil", err)
	}
	for _, sp := range set.Specs {
		changed := *sp.Model
		changed.Endpoint.HostSuffix = []string{".wiki.example.org"}
		if issues := conformance.CheckHosts(&changed); len(issues) == 0 {
			t.Fatalf("CheckHosts() of %s with an outside host = none, want an issue", sp.File)
		}
	}
}

// TestBundledKindsStaySorted proves the enumeration order the goldens assume:
// the bundled specs load in embedded name order.
func TestBundledKindsStaySorted(t *testing.T) {
	set, err := connectorspec.LoadBundled()
	if err != nil {
		t.Fatalf("LoadBundled() error = %v, want nil", err)
	}
	files := make([]string, 0, len(set.Specs))
	for _, sp := range set.Specs {
		files = append(files, sp.File)
	}
	if !slices.IsSorted(files) {
		t.Fatalf("bundled specs = %v, want embedded name order", files)
	}
}
