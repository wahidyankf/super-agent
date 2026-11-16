package main

import (
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/jiraseam"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestReadSeamRefusesAJiraSeamWithoutItsCAFile checks the error path of the Jira seam: a base without the CA file is
// refused, the same rule as the Outline seam.
func TestReadSeamRefusesAJiraSeamWithoutItsCAFile(t *testing.T) {
	getenv := mapEnv(map[string]string{jiraseam.BaseURLEnv: "https://127.0.0.1:1"})
	if _, err := readSeam(domain.KindJira, getenv); err == nil {
		t.Fatal("readSeam accepted a Jira seam without its CA file")
	}
}

// TestRegularFileExistsReportsOnlyRegularFiles checks the path test: a missing
// path is not a regular file, and neither is a folder.
func TestRegularFileExistsReportsOnlyRegularFiles(t *testing.T) {
	if regularFileExists("definitely-not-a-retriever-file") {
		t.Fatal("regularFileExists() of a missing path is true")
	}
	if regularFileExists(".") {
		t.Fatal("regularFileExists() of a folder is true")
	}
}

// TestRealDirExistsReportsOnlyRealFolders checks the folder check the profile resolution uses: a missing path and a
// file are not folders, and neither is a folder reached through a link. The check describes the link itself, following
// nothing, so it runs against the working tree and reads no profile.
func TestRealDirExistsReportsOnlyRealFolders(t *testing.T) {
	if !realDirExists(".") {
		t.Fatal("realDirExists() of a folder is false")
	}
	if realDirExists("definitely-not-a-retriever-file") {
		t.Fatal("realDirExists() of a missing path is true")
	}
	if realDirExists("disk_open_test.go") {
		t.Fatal("realDirExists() of a file is true")
	}
}

// TestRealSymlinkExistsReportsOnlyLinks checks the link check the profile resolution uses: a folder, a file, and a
// missing path are not links.
func TestRealSymlinkExistsReportsOnlyLinks(t *testing.T) {
	if realSymlinkExists(".") {
		t.Fatal("realSymlinkExists() of a folder is true")
	}
	if realSymlinkExists("disk_open_test.go") {
		t.Fatal("realSymlinkExists() of a file is true")
	}
	if realSymlinkExists("definitely-not-a-retriever-file") {
		t.Fatal("realSymlinkExists() of a missing path is true")
	}
}
