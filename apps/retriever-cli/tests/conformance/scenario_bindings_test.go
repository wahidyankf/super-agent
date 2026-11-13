//go:build bdd

// Run by the bdd:check target (go test -tags bdd), not by test:unit: a binding gap is a gap in the specs, not in the code.
package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// This file is the binding check of specs/behaviours (the bdd:check equivalent). Every scenario carries a unit binding,
// and an integration and an end-to-end binding unless it carries the matching exempt tag. An exempt tag needs a comment
// directly above the tag line, in the form "# Exemption(<level>): <why>; alternative-proof: <where>". A scenario with
// both exempt tags is a detail and belongs in a plain test. The file behaviour-coverage.json must match the feature files.

const (
	specsRoot    = "../../../../specs/behaviours"
	coverageFile = "../../behaviour-coverage.json"
)

var exemptionLine = regexp.MustCompile(`^#\s*Exemption\((integration|e2e)\):\s*\S.*;\s*alternative-proof:\s*\S`)

type scenarioTags struct {
	file, title string
	line        int
	tags        map[string]bool
	exemptions  map[string]bool
}

// readScenarios returns every Scenario and Scenario Outline of one feature file, with its tags and justified exemptions.
func readScenarios(t *testing.T, path string) []scenarioTags {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out []scenarioTags
	var tags []string
	pendingExemptions := map[string]bool{}
	for i, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "@"):
			tags = append(tags, strings.Fields(trimmed)...)
		case strings.HasPrefix(trimmed, "#"):
			if m := exemptionLine.FindStringSubmatch(trimmed); m != nil {
				pendingExemptions[m[1]] = true
			}
		case strings.HasPrefix(trimmed, "Scenario:") || strings.HasPrefix(trimmed, "Scenario Outline:"):
			sc := scenarioTags{file: path, title: trimmed, line: i + 1, tags: map[string]bool{}, exemptions: pendingExemptions}
			for _, tag := range tags {
				sc.tags[tag] = true
			}
			out = append(out, sc)
			tags, pendingExemptions = nil, map[string]bool{}
		default:
			tags, pendingExemptions = nil, map[string]bool{}
		}
	}
	return out
}

func featureFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(specsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".feature") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", specsRoot, err)
	}
	return files
}

// bindingProblems returns one message for each binding gap in the given feature files. An empty result means the files
// conform. The check does not stop at the first gap, so one run reports every gap.
func bindingProblems(t *testing.T, files []string) []string {
	t.Helper()
	var problems []string
	for _, file := range files {
		for _, sc := range readScenarios(t, file) {
			where := sc.file + ":" + strconv.Itoa(sc.line) + " " + sc.title
			if !sc.tags["@unit"] {
				problems = append(problems, where+": no unit binding (@unit); the unit binding is always required")
			}
			if sc.tags["@integration-exempt"] && sc.tags["@e2e-exempt"] {
				problems = append(problems, where+": exempt at both layers (@integration-exempt and @e2e-exempt); a scenario with both exemptions is a plain test, not a spec scenario")
			}
			for _, level := range []string{"integration", "e2e"} {
				tag := "@" + level
				exempt := "@" + level + "-exempt"
				switch {
				case sc.tags[tag] && sc.tags[exempt]:
					problems = append(problems, where+": carries both "+tag+" and "+exempt)
				case sc.tags[tag]:
				case sc.tags[exempt] && sc.exemptions[level]:
				case sc.tags[exempt]:
					problems = append(problems, where+": "+exempt+" has no Exemption("+level+") comment with an alternative-proof directly above the tags")
				default:
					problems = append(problems, where+": no "+tag+" binding and no "+exempt+" tag")
				}
			}
		}
	}
	return problems
}

// TestEveryScenarioHasItsBindingsOrAJustifiedExemption fails for each scenario that lacks a unit binding, an integration
// or end-to-end binding without a justified exemption, or both exemptions at once.
func TestEveryScenarioHasItsBindingsOrAJustifiedExemption(t *testing.T) {
	files := featureFiles(t)
	if len(files) == 0 {
		t.Fatalf("no feature files under %s", specsRoot)
	}
	for _, problem := range bindingProblems(t, files) {
		t.Error(problem)
	}
}

// writeCorpus writes one feature file into dir, with the given text, and returns its path.
func writeCorpus(t *testing.T, dir, name, text string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestBindingCheckFailsOnViolatingCorpus runs the check on a synthetic corpus of feature files that each break one rule.
// The check must report every violation, each with the scenario it names.
func TestBindingCheckFailsOnViolatingCorpus(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeCorpus(t, dir, "missing-unit.feature", `Feature: synthetic
  @integration @e2e
  Scenario: lacks the unit binding
    When something runs
`),
		writeCorpus(t, dir, "missing-comment.feature", `Feature: synthetic
  @unit @integration-exempt @e2e
  Scenario: exempt without the comment
    When something runs
`),
		writeCorpus(t, dir, "malformed-comment.feature", `Feature: synthetic
  # Exemption(e2e): no alternative proof named
  @unit @integration @e2e-exempt
  Scenario: exempt with a malformed comment
    When something runs
`),
		writeCorpus(t, dir, "both-exempt.feature", `Feature: synthetic
  # Exemption(integration): synthetic reason; alternative-proof: synthetic location
  # Exemption(e2e): synthetic reason; alternative-proof: synthetic location
  @unit @integration-exempt @e2e-exempt
  Scenario: exempt at both layers
    When something runs
`),
	}
	problems := bindingProblems(t, files)
	want := []struct{ scenario, fragment string }{
		{"lacks the unit binding", "no unit binding (@unit)"},
		{"exempt without the comment", "@integration-exempt has no Exemption(integration) comment"},
		{"exempt with a malformed comment", "@e2e-exempt has no Exemption(e2e) comment"},
		{"exempt at both layers", "exempt at both layers"},
	}
	for _, w := range want {
		if !containsProblem(problems, w.scenario, w.fragment) {
			t.Errorf("the check did not report %q for scenario %q; problems were:\n%s", w.fragment, w.scenario, strings.Join(problems, "\n"))
		}
	}
	if len(problems) != len(want) {
		t.Errorf("the check reported %d problems, want %d, one per violating scenario:\n%s", len(problems), len(want), strings.Join(problems, "\n"))
	}
}

// TestBindingCheckPassesOnConformingCorpus runs the check on feature files that follow every rule, and expects none.
func TestBindingCheckPassesOnConformingCorpus(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeCorpus(t, dir, "bound.feature", `Feature: synthetic
  @unit @integration @e2e
  Scenario: fully bound
    When something runs
`),
		writeCorpus(t, dir, "exempt-e2e.feature", `Feature: synthetic
  # Exemption(e2e): synthetic reason; alternative-proof: synthetic location
  @unit @integration @e2e-exempt
  Scenario: exempt from the end-to-end level with its proof
    When something runs
`),
	}
	if problems := bindingProblems(t, files); len(problems) != 0 {
		t.Errorf("a conforming corpus must pass, got:\n%s", strings.Join(problems, "\n"))
	}
}

// containsProblem reports whether one problem names the scenario and contains the fragment.
func containsProblem(problems []string, scenario, fragment string) bool {
	for _, p := range problems {
		if strings.Contains(p, scenario) && strings.Contains(p, fragment) {
			return true
		}
	}
	return false
}

// TestBehaviourCoverageFileMatchesTheFeatures compares the counts in behaviour-coverage.json with the feature files.
func TestBehaviourCoverageFileMatchesTheFeatures(t *testing.T) {
	type counts struct {
		Path        string `json:"path"`
		Scenarios   int    `json:"scenarios"`
		Unit        int    `json:"unit"`
		Integration int    `json:"integration"`
		E2E         int    `json:"e2e"`
	}
	var doc struct {
		Features []counts `json:"features"`
		Totals   struct {
			Scenarios, Unit, Integration, E2E int
		} `json:"totals"`
	}
	raw, err := os.ReadFile(coverageFile)
	if err != nil {
		t.Fatalf("read %s: %v", coverageFile, err)
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse %s: %v", coverageFile, err)
	}

	recorded := map[string]counts{}
	for _, f := range doc.Features {
		recorded[f.Path] = f
	}
	var total counts
	for _, file := range featureFiles(t) {
		var got counts
		got.Path = filepath.ToSlash(filepath.Join("specs/behaviours", strings.TrimPrefix(file, specsRoot+"/")))
		for _, sc := range readScenarios(t, file) {
			got.Scenarios++
			if sc.tags["@unit"] {
				got.Unit++
			}
			if sc.tags["@integration"] {
				got.Integration++
			}
			if sc.tags["@e2e"] {
				got.E2E++
			}
		}
		total.Scenarios += got.Scenarios
		total.Unit += got.Unit
		total.Integration += got.Integration
		total.E2E += got.E2E
		want, ok := recorded[got.Path]
		if !ok {
			t.Errorf("%s is missing from %s", got.Path, coverageFile)
			continue
		}
		if want != got {
			t.Errorf("%s: %s records %+v, the feature holds %+v", got.Path, coverageFile, want, got)
		}
	}
	if doc.Totals.Scenarios != total.Scenarios || doc.Totals.Unit != total.Unit ||
		doc.Totals.Integration != total.Integration || doc.Totals.E2E != total.E2E {
		t.Errorf("totals in %s are %+v, the feature files hold %+v", coverageFile, doc.Totals, total)
	}
}
