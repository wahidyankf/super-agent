//go:build integration

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
)

// envOf returns a lookup over the given values, so a test sets variables without changing the process environment.
func envOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
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
	err := attachSourcesFrom(&deps, cli.Globals{Catalog: catalog, EnvFile: filepath.Join(t.TempDir(), ".env.local")},
		seamSet{}, emptyEnv, os.Getwd)
	if err == nil || !strings.Contains(err.Error(), "sources catalog: not found") {
		t.Fatalf("attachSourcesFrom error = %v, want the missing-catalog error", err)
	}
}

// TestAttachSourcesWiresTheExampleCatalog checks that the example catalog wires every source the CLI and the servers need.
func TestAttachSourcesWiresTheExampleCatalog(t *testing.T) {
	var deps cli.Deps
	err := attachSourcesFrom(&deps, cli.Globals{Catalog: filepath.Join("..", "..", "examples", "sources.example.yml"),
		EnvFile: filepath.Join(t.TempDir(), ".env.local")}, seamSet{}, emptyEnv, os.Getwd)
	if err != nil {
		t.Fatalf("attachSourcesFrom: %v", err)
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
