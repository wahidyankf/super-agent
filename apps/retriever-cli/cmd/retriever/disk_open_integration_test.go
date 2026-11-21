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

// TestRealDiskPathChecksAgreeWithTheWorkingTree checks the three path checks against the real disk: a missing path is none
// of them, a folder is a folder and not a file or a link, and a file is a file. The checks read the disk through their
// stat seams, and the default seams are the real calls, so this test is the one that reads the working tree.
func TestRealDiskPathChecksAgreeWithTheWorkingTree(t *testing.T) {
	if regularFileExists("definitely-not-a-retriever-file") || regularFileExists(".") {
		t.Fatal("regularFileExists accepts a missing path or a folder on the real disk")
	}
	if !realDirExists(".") || realDirExists("definitely-not-a-retriever-file") || realDirExists("disk_open_test.go") {
		t.Fatal("realDirExists accepts only the working folder on the real disk")
	}
	if realSymlinkExists(".") || realSymlinkExists("disk_open_test.go") || realSymlinkExists("definitely-not-a-retriever-file") {
		t.Fatal("realSymlinkExists accepts a folder, a file, or a missing path on the real disk")
	}
}
