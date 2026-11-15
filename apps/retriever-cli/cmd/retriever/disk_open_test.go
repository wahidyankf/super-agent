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
