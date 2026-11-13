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

// TestEveryScenarioHasItsBindingsOrAJustifiedExemption fails for each scenario that lacks a unit binding, an integration
// or end-to-end binding without a justified exemption, or both exemptions at once.
func TestEveryScenarioHasItsBindingsOrAJustifiedExemption(t *testing.T) {
	files := featureFiles(t)
	if len(files) == 0 {
		t.Fatalf("no feature files under %s", specsRoot)
	}
	for _, file := range files {
		for _, sc := range readScenarios(t, file) {
			where := sc.file + ":" + strconv.Itoa(sc.line) + " " + sc.title
			if !sc.tags["@unit"] {
				t.Errorf("%s: no unit binding (@unit); the unit binding is always required", where)
			}
			for _, level := range []string{"integration", "e2e"} {
				tag := "@" + level
				exempt := "@" + level + "-exempt"
				switch {
				case sc.tags[tag] && sc.tags[exempt]:
					t.Errorf("%s: carries both %s and %s", where, tag, exempt)
				case sc.tags[tag]:
				case sc.tags[exempt] && sc.exemptions[level]:
				case sc.tags[exempt]:
					t.Errorf("%s: %s has no Exemption(%s) comment with an alternative-proof directly above the tags", where, exempt, level)
				default:
					t.Errorf("%s: no %s binding and no %s tag", where, tag, exempt)
				}
			}
		}
	}
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
