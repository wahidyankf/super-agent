//go:build integration

package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file keeps the one binding rule that RHINO's behaviour coverage rules (B01 to B09) do not state: a scenario must
// not be exempt at both the integration and the end-to-end level. A scenario with both exemptions is a plain test, not
// a spec scenario. The rule is proposed to RHINO as a new behaviour rule; until RHINO states it, this test enforces it.
// It carries the integration tag, because it reads the checked-in corpus from disk (D10); test:conformance runs it.

const exemptionsSpecsRoot = "../../../../specs/behaviours"

// scenariosExemptAtBothLayers returns the title of each scenario of one feature file that carries both exempt tags.
// The tags of a scenario are the @ lines directly above its Scenario keyword; comment lines keep them, and any other
// line resets them.
func scenariosExemptAtBothLayers(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var found []string
	var tags []string
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "@"):
			tags = append(tags, strings.Fields(trimmed)...)
		case strings.HasPrefix(trimmed, "#"):
		case strings.HasPrefix(trimmed, "Scenario:") || strings.HasPrefix(trimmed, "Scenario Outline:"):
			if hasTag(tags, "@integration-exempt") && hasTag(tags, "@e2e-exempt") {
				found = append(found, path+": "+trimmed)
			}
			tags = nil
		default:
			tags = nil
		}
	}
	return found
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

// TestNoScenarioIsExemptAtBothLayers fails for each scenario of the behaviour corpus that is exempt at both levels.
func TestNoScenarioIsExemptAtBothLayers(t *testing.T) {
	var features []string
	err := filepath.Walk(exemptionsSpecsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".feature") {
			features = append(features, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", exemptionsSpecsRoot, err)
	}
	if len(features) == 0 {
		t.Fatalf("no feature files under %s", exemptionsSpecsRoot)
	}
	for _, feature := range features {
		for _, scenario := range scenariosExemptAtBothLayers(t, feature) {
			t.Errorf("%s: exempt at both layers (@integration-exempt and @e2e-exempt); a scenario with both exemptions is a plain test, not a spec scenario", scenario)
		}
	}
}

// TestBothExemptionsCheckFindsThePair runs the check on a synthetic feature that carries both exempt tags, and expects
// the scenario named. It also expects a scenario with one exemption to pass.
func TestBothExemptionsCheckFindsThePair(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pair.feature")
	text := `Feature: synthetic
  # Exemption(integration): synthetic reason; alternative-proof: synthetic location
  @integration-exempt
  # Exemption(e2e): synthetic reason; alternative-proof: synthetic location
  @e2e-exempt
  Scenario: exempt at both layers
    When something runs

  # Exemption(e2e): synthetic reason; alternative-proof: synthetic location
  @e2e-exempt
  Scenario: exempt at one layer
    When something runs
`
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	found := scenariosExemptAtBothLayers(t, path)
	if len(found) != 1 || !strings.Contains(found[0], "exempt at both layers") {
		t.Errorf("the check found %v, want the one scenario exempt at both layers", found)
	}
}
