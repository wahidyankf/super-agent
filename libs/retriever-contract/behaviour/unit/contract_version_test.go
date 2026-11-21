package unit_test

import (
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract"
)

// @scenario specs/libs/retriever-contract/behaviours/contract-version.feature: The contract reports version 0.4.0
func TestContractReportsVersion040(t *testing.T) {
	if got := contract.ContractVersion; got != "0.4.0" {
		t.Errorf("version = %q, want %q", got, "0.4.0")
	}
}
