package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
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

// TestRegularFileExistsIsFalseForADirectoryAndTrueForAFile checks the existence rule the path resolution uses.
func TestRegularFileExistsIsFalseForADirectoryAndTrueForAFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sources.yml")
	if regularFileExists(dir) || regularFileExists(file) {
		t.Fatal("a directory or a missing file must not count as a regular file")
	}
	if err := os.WriteFile(file, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !regularFileExists(file) {
		t.Fatal("a regular file must count as present")
	}
}

// TestAttachSourcesReportsAMissingCatalog checks that an absent catalog is a setup error, so only the sources commands fail.
func TestAttachSourcesReportsAMissingCatalog(t *testing.T) {
	var deps cli.Deps
	catalog := filepath.Join(t.TempDir(), "sources.yml")
	err := attachSources(&deps, cli.Globals{Catalog: catalog, EnvFile: filepath.Join(t.TempDir(), ".env.local")}, seamSet{})
	if err == nil || !strings.Contains(err.Error(), "sources catalog: not found") {
		t.Fatalf("attachSources error = %v, want the missing-catalog error", err)
	}
}

// TestAttachSourcesWiresTheExampleCatalog checks that the example catalog wires every source the CLI and the servers need.
func TestAttachSourcesWiresTheExampleCatalog(t *testing.T) {
	var deps cli.Deps
	err := attachSources(&deps, cli.Globals{Catalog: filepath.Join("..", "..", "examples", "sources.example.yml"),
		EnvFile: filepath.Join(t.TempDir(), ".env.local")}, seamSet{})
	if err != nil {
		t.Fatalf("attachSources: %v", err)
	}
	if got := len(deps.Service.Catalog.All()); got != 3 {
		t.Fatalf("the example catalog wires %d sources, want 3", got)
	}
	if deps.Serve == nil || deps.MCP == nil {
		t.Fatal("the serve and MCP functions must be wired")
	}
}

// TestExportSinkWritesIntoTheGivenFolder checks that the sink the CLI gets writes its file into the folder the flag named.
func TestExportSinkWritesIntoTheGivenFolder(t *testing.T) {
	dir := t.TempDir()
	location, err := exportSink(dir).Write(t.Context(), "example.md", []byte("# Example\n"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if location != filepath.Join(dir, "example.md") {
		t.Fatalf("location = %q, want the file under %q", location, dir)
	}
	if data, err := os.ReadFile(location); err != nil || string(data) != "# Example\n" {
		t.Fatalf("file = %q (%v), want the Markdown", data, err)
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
