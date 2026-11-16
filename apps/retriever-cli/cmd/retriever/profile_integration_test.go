//go:build integration

package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// profileTree writes a profiles folder under home with a work and a personal profile, each holding a catalog with one
// Outline source whose credential carries the profile prefix.
func profileTree(t *testing.T, home string) {
	t.Helper()
	write := func(profile, id, credential string) {
		dir := filepath.Join(home, ".config", "retriever", "profiles", profile)
		if err := os.MkdirAll(filepath.Join(dir, "connectors"), 0o755); err != nil {
			t.Fatal(err)
		}
		catalogYAML := "version: 1\nsources:\n  - id: " + id + "\n    kind: outline\n" +
			"    endpoint: https://docs.example.test\n    credential_env: " + credential + "\n"
		if err := os.WriteFile(filepath.Join(dir, "sources.yml"), []byte(catalogYAML), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte(credential+"=synthetic-key\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("work", "work-docs", "WORK_OUTLINE_API_KEY")
	write("personal", "personal-docs", "PERSONAL_OUTLINE_API_KEY")
}

// profileEnv points HOME at home and clears every path variable, so the run resolves the profiles folder under home.
func profileEnv(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("RETRIEVER_PROFILE", "")
	t.Setenv("RETRIEVER_CATALOG", "")
	t.Setenv("RETRIEVER_ENV_FILE", "")
}

// recordingProfileHost returns a host over the real disk that records every catalog, env file, and connectors folder
// the wiring opens, so the test proves which profile folder a run reads.
func recordingProfileHost(home string, opened *[]string) host {
	record := func(kind, path string) { *opened = append(*opened, kind+":"+path) }
	h := realHost
	h.stdin = strings.NewReader("")
	h.stdout = &bytes.Buffer{}
	h.stderr = &bytes.Buffer{}
	h.getwd = func() (string, error) { return home, nil }
	h.openCatalog = func(path string, kinds domain.KindSet) (*catalog.Loader, error) {
		record("catalog", path)
		return openCatalog(path, kinds)
	}
	h.openEnvFile = func(path string) (*envfile.Credentials, error) {
		record("env", path)
		return openEnvFile(path)
	}
	h.openConnectors = func(dir string) fs.FS {
		record("connectors", dir)
		return openConnectorsFolder(dir)
	}
	return h
}

// profileOut runs argv over a recording host and returns the status and the two streams.
func profileOut(t *testing.T, argv []string, h host) (int, string, string) {
	t.Helper()
	status := run(argv, os.Getenv, h)
	return status, h.stdout.(*bytes.Buffer).String(), h.stderr.(*bytes.Buffer).String()
}

// TestPersonalRunReadsNoWorkFolder checks F1 end to end: with work and personal profiles present, a personal run lists
// its own source, names its profile, and the recording file system shows no read of the work folder.
func TestPersonalRunReadsNoWorkFolder(t *testing.T) {
	home := t.TempDir()
	profileTree(t, home)
	profileEnv(t, home)
	var opened []string
	h := recordingProfileHost(home, &opened)

	status, out, errOut := profileOut(t, []string{"--profile", "personal", "sources", "list"}, h)
	if status != 0 {
		t.Fatalf("run(sources list) = %d, want 0: %s", status, errOut)
	}
	if !strings.Contains(out, "personal-docs") || !strings.Contains(out, "(profile: personal)") {
		t.Fatalf("sources list = %q, want the personal source and profile", out)
	}
	if strings.Contains(out, "work-docs") {
		t.Fatalf("sources list = %q, must not name the work source", out)
	}
	if len(opened) == 0 {
		t.Fatal("the run opened no file, want the personal folder")
	}
	for _, entry := range opened {
		if !strings.Contains(entry, "/profiles/personal/") {
			t.Fatalf("the personal run opened %q, want only the personal folder: %q", entry, opened)
		}
		if strings.Contains(entry, "/profiles/work") {
			t.Fatalf("the personal run read the work folder: %q", entry)
		}
	}
}

// TestPersonalCatalogWithAWorkVariableFails checks F1 at load end to end: a personal catalog that names a
// work-prefixed variable fails with the prefix rule, exits 2, and keeps standard output empty.
func TestPersonalCatalogWithAWorkVariableFails(t *testing.T) {
	home := t.TempDir()
	profileTree(t, home)
	profileEnv(t, home)
	personal := filepath.Join(home, ".config", "retriever", "profiles", "personal", "sources.yml")
	crossed, err := os.ReadFile(personal)
	if err != nil {
		t.Fatal(err)
	}
	crossed = []byte(strings.Replace(string(crossed), "PERSONAL_OUTLINE_API_KEY", "WORK_OUTLINE_API_KEY", 1))
	if err := os.WriteFile(personal, crossed, 0o600); err != nil {
		t.Fatal(err)
	}
	var opened []string
	h := recordingProfileHost(home, &opened)

	status, out, errOut := profileOut(t, []string{"--output", "json", "--profile", "personal", "sources", "list"}, h)
	if status != 2 || out != "" {
		t.Fatalf("run(sources list) = %d with stdout %q, want 2 with empty stdout", status, out)
	}
	if !strings.Contains(errOut, "must start with the profile prefix") {
		t.Fatalf("stderr = %q, want the prefix rule", errOut)
	}
}

// TestRealDirAndSymlinkChecks checks the folder checks the profile resolution uses: a folder counts, a file and a
// symbolic link do not, and only the link reports as a link.
func TestRealDirAndSymlinkChecks(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sources.yml")
	if err := os.WriteFile(file, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if !realDirExists(dir) || realDirExists(file) || realDirExists(link) || realDirExists(filepath.Join(dir, "missing")) {
		t.Fatal("realDirExists must accept only the folder")
	}
	if realSymlinkExists(dir) || realSymlinkExists(file) || !realSymlinkExists(link) {
		t.Fatal("realSymlinkExists must accept only the link")
	}
}
