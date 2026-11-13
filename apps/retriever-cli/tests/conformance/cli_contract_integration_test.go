//go:build integration

package conformance

import (
	"encoding/json"
	"os"
	"testing"
)

// loadFixture reads and decodes the contract fixture.
func loadFixture(t *testing.T) fixture {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read the conformance fixture: %v", err)
	}
	var fx fixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("decode the conformance fixture: %v", err)
	}
	return fx
}

// TestContractChecks runs every published check. A check that fails is a failure, unless the fixture lists it under
// knownGaps. A listed gap that passes is also a failure, so a fixed gap must be removed from the list.
func TestContractChecks(t *testing.T) {
	fx := loadFixture(t)
	if fx.SchemaVersion != 1 {
		t.Fatalf("the fixture schemaVersion is %d, want 1", fx.SchemaVersion)
	}
	closed := map[int]bool{}
	for _, status := range fx.ClosedExitStatuses {
		closed[status] = true
	}
	ids := map[string]bool{}
	for _, a := range fx.Assertions {
		ids[a.ID] = true
	}
	gaps := map[string]bool{}
	for _, g := range fx.KnownGaps {
		if !ids[g] {
			t.Errorf("knownGaps names %q, which no assertion has", g)
		}
		gaps[g] = true
	}
	for _, a := range fx.Assertions {
		t.Run(a.ID, func(t *testing.T) {
			if !closed[a.ExitStatus] {
				t.Fatalf("exit status %d is outside the closed set %v", a.ExitStatus, fx.ClosedExitStatuses)
			}
			fails := runCheck(a)
			if gaps[a.ID] {
				if len(fails) == 0 {
					t.Errorf("%s is listed in knownGaps but passes; remove it from the list", a.ID)
				}
				return
			}
			for _, f := range fails {
				t.Error(f)
			}
		})
	}
}
