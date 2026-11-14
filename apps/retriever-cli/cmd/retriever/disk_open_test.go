package main

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-jira/jira"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestReadSeamRefusesAJiraSeamWithoutItsCAFile checks the error path of the Jira seam: a base without the CA file is
// refused, the same rule as the Outline seam.
func TestReadSeamRefusesAJiraSeamWithoutItsCAFile(t *testing.T) {
	getenv := mapEnv(map[string]string{jira.BaseURLEnv: "https://127.0.0.1:1"})
	if _, err := readSeam(domain.KindJira, getenv); err == nil {
		t.Fatal("readSeam accepted a Jira seam without its CA file")
	}
}
