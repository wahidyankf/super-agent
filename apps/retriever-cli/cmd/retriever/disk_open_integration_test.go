//go:build integration

package main

import "testing"

// TestOpenersBuildTheReadersWithoutReadingTheDisk checks the real openers on absolute paths. Building a reader or a loader
// reads nothing: the file is read only when a lookup or a load runs, so this test touches no file. The real openers call
// package os, so the test carries the integration tag.
func TestOpenersBuildTheReadersWithoutReadingTheDisk(t *testing.T) {
	if _, err := openCatalog("/work/project/sources.yml", registeredKinds()); err != nil {
		t.Fatalf("openCatalog error = %v, want nil", err)
	}
	if _, err := openEnvFile("/work/project/.env.local"); err != nil {
		t.Fatalf("openEnvFile error = %v, want nil", err)
	}
}
