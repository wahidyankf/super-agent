package main

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-connector-outline/outline"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestReadSeamRefusesAnOutlineSeamWithoutItsCAFile checks the error path of one kind's seam: an Outline base without the
// CA file that the seam needs is refused, so no seam falls back to a production host.
func TestReadSeamRefusesAnOutlineSeamWithoutItsCAFile(t *testing.T) {
	getenv := mapEnv(map[string]string{outline.BaseURLEnv: "https://127.0.0.1:1"})
	if _, err := readSeam(domain.KindOutline, getenv); err == nil {
		t.Fatal("readSeam accepted an Outline seam without its CA file")
	}
}
