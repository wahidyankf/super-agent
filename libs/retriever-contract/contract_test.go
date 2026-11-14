package contract

import "testing"

// TestContractVersionMatchesTheLockstepRelease checks the version the contract reports. A change to the release line
// must change this test too.
func TestContractVersionMatchesTheLockstepRelease(t *testing.T) {
	if ContractVersion != "0.3.0" {
		t.Errorf("ContractVersion = %q, want %q", ContractVersion, "0.3.0")
	}
}
