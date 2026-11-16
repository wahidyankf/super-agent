package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
)

// personalCatalogYAML is a catalog with one Outline source whose credential carries the personal prefix.
const personalCatalogYAML = `version: 1
sources:
  - id: docs
    kind: outline
    endpoint: https://docs.example.test
    credential_env: PERSONAL_OUTLINE_API_KEY
`

// profileGetenv returns a lookup over the given values, with HOME under /cfg, so the profiles folder
// (/cfg/.config/retriever/profiles) is fixed without the real disk.
func profileGetenv(values map[string]string) func(string) string {
	return func(key string) string {
		if key == "HOME" {
			return "/cfg"
		}
		return values[key]
	}
}

// profileTestHost returns the unit-test host whose profiles folder holds the work and personal profiles, and whose
// catalog file holds catalogYAML. The catalog path the wiring opens is recorded in gotCatalog.
func profileTestHost(catalogYAML string, gotCatalog *string) host {
	files := fstest.MapFS{"sources.yml": &fstest.MapFile{Data: []byte(catalogYAML)}}
	h := testHost()
	h.isDir = func(path string) bool { return path == "/cfg/.config/retriever/profiles" }
	h.isSymlink = func(string) bool { return false }
	h.openCatalog = func(path string, kinds domain.KindSet) (*catalog.Loader, error) {
		*gotCatalog = path
		return catalog.NewWithFiles(path, catalog.Files{FS: files, SameFile: sameName}, kinds)
	}
	return h
}

// profileCode returns the stable code of a setup error that carries its own code.
func profileCode(err error) string {
	type coder interface{ ErrorCode() string }
	if coded, ok := err.(coder); ok {
		return coded.ErrorCode()
	}
	return ""
}

// TestAttachSourcesSelectsTheProfileFolder checks that a personal run resolves the personal catalog, and that the
// service names the profile the sources list prints.
func TestAttachSourcesSelectsTheProfileFolder(t *testing.T) {
	var deps cli.Deps
	var gotCatalog string
	h := profileTestHost(personalCatalogYAML, &gotCatalog)
	globals := cli.Globals{Profile: "personal"}
	if err := attachSourcesFrom(&deps, globals, seamSet{}, profileGetenv(nil), h); err != nil {
		t.Fatalf("attachSourcesFrom: %v", err)
	}
	if want := "/cfg/.config/retriever/profiles/personal/sources.yml"; gotCatalog != want {
		t.Fatalf("catalog path = %q, want %q", gotCatalog, want)
	}
	if deps.Service.Profile != "personal" {
		t.Fatalf("service profile = %q, want personal", deps.Service.Profile)
	}
}

// TestAttachSourcesRefusesProfilesWithNoChoice checks F3: profiles exist, none is chosen, and the setup error carries
// the profile_required code before any catalog is opened.
func TestAttachSourcesRefusesProfilesWithNoChoice(t *testing.T) {
	var deps cli.Deps
	var gotCatalog string
	h := profileTestHost(personalCatalogYAML, &gotCatalog)
	if err := attachSourcesFrom(&deps, cli.Globals{}, seamSet{}, profileGetenv(nil), h); profileCode(err) !=
		"retriever.usage.profile_required" {
		t.Fatalf("attachSourcesFrom error = %v, want the profile_required code", err)
	}
	if gotCatalog != "" {
		t.Fatalf("catalog opened at %q, want no catalog read", gotCatalog)
	}
}

// TestAttachSourcesRefusesAnExplicitCatalogWithAProfile checks F2: a profile together with --catalog is a usage error.
func TestAttachSourcesRefusesAnExplicitCatalogWithAProfile(t *testing.T) {
	var deps cli.Deps
	var gotCatalog string
	h := profileTestHost(personalCatalogYAML, &gotCatalog)
	globals := cli.Globals{Profile: "personal", Catalog: "/elsewhere/sources.yml"}
	if err := attachSourcesFrom(&deps, globals, seamSet{}, profileGetenv(nil), h); profileCode(err) !=
		"retriever.usage.profile_invalid" {
		t.Fatalf("attachSourcesFrom error = %v, want the profile_invalid code", err)
	}
}

// TestRunProfilesWithNoChoiceReportsProfileRequired checks the gate: profiles exist, no profile is chosen, and any
// source command exits 2 with retriever.usage.profile_required and empty standard output.
func TestRunProfilesWithNoChoiceReportsProfileRequired(t *testing.T) {
	var gotCatalog string
	h := profileTestHost(personalCatalogYAML, &gotCatalog)
	argv := []string{"--output", "json", "sources", "list"}
	if status := run(argv, profileGetenv(nil), h); status != 2 {
		t.Fatalf("run(sources list) = %d, want 2", status)
	}
	if out := h.stdout.(*bytes.Buffer).String(); out != "" {
		t.Fatalf("standard output = %q, want empty", out)
	}
	if errOut := h.stderr.(*bytes.Buffer).String(); !strings.Contains(errOut, `"code":"retriever.usage.profile_required"`) {
		t.Fatalf("standard error = %q, want the profile_required code", errOut)
	}
}

// TestRunProfileTraversalExitsTwo checks the gate: --profile ../other exits 2, because the name escapes its folder.
func TestRunProfileTraversalExitsTwo(t *testing.T) {
	var gotCatalog string
	h := profileTestHost(personalCatalogYAML, &gotCatalog)
	argv := []string{"--profile", "../other", "sources", "list"}
	if status := run(argv, profileGetenv(nil), h); status != 2 {
		t.Fatalf("run(sources list) = %d, want 2", status)
	}
	if out := h.stdout.(*bytes.Buffer).String(); out != "" {
		t.Fatalf("standard output = %q, want empty", out)
	}
}

// TestRunPersonalSourcesListNamesTheProfile checks F4 end to end: a personal run lists its sources and names the
// active profile.
func TestRunPersonalSourcesListNamesTheProfile(t *testing.T) {
	var gotCatalog string
	h := profileTestHost(personalCatalogYAML, &gotCatalog)
	argv := []string{"--profile", "personal", "sources", "list"}
	if status := run(argv, profileGetenv(nil), h); status != 0 {
		t.Fatalf("run(sources list) = %d, want 0: %s", status, h.stderr.(*bytes.Buffer).String())
	}
	if out := h.stdout.(*bytes.Buffer).String(); !strings.Contains(out, "(profile: personal)") {
		t.Fatalf("sources list = %q, want the personal profile", out)
	}
}
