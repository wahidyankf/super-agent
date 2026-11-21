//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
)

// isolatedEnv points HOME at home and clears the variables the rules read, so the run sees only what the test sets.
func isolatedEnv(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	for _, name := range []string{"XDG_CONFIG_HOME", "RETRIEVER_CATALOG", "RETRIEVER_ENV_FILE", "RETRIEVER_PROFILE"} {
		unsetEnv(t, name)
	}
}

// realInputs returns the resolver inputs over the real environment, working directory, and file metadata. Each check
// follows no symbolic link, so a link is seen as a link.
func realInputs() configpaths.Inputs {
	return configpaths.Inputs{
		Getenv: os.Getenv,
		Getwd:  os.Getwd,
		Exists: func(path string) bool {
			info, err := os.Lstat(path)
			return err == nil && info.Mode().IsRegular()
		},
		IsDir: func(path string) bool {
			info, err := os.Lstat(path)
			return err == nil && info.IsDir()
		},
		IsSymlink: func(path string) bool {
			info, err := os.Lstat(path)
			return err == nil && info.Mode()&os.ModeSymlink != 0
		},
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: The first rule that applies sets the catalog path
func TestTheFirstRuleThatAppliesSetsTheCatalogPath(t *testing.T) {
	cases := []struct {
		name  string
		flag  string
		env   string
		atCwd bool
		path  func(home, wd string) string
	}{
		{"the flag wins", "/flag/sources.yml", "/env/sources.yml", true,
			func(string, string) string { return "/flag/sources.yml" }},
		{"the variable is next", "", "/env/sources.yml", true,
			func(string, string) string { return "/env/sources.yml" }},
		{"the working directory file is next", "", "", true,
			func(_, wd string) string { return filepath.Join(wd, "sources.yml") }},
		{"the user folder is last", "", "", false,
			func(home, _ string) string { return filepath.Join(home, ".config", "retriever", "sources.yml") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			wd := t.TempDir()
			isolatedEnv(t, home)
			t.Chdir(wd)
			if tc.env != "" {
				t.Setenv("RETRIEVER_CATALOG", tc.env)
			}
			if tc.atCwd {
				writeFiles(t, wd, map[string]string{"sources.yml": outlineDocsYAML})
			}
			in := realInputs()
			in.CatalogFlag = tc.flag
			paths, err := configpaths.Resolve(in)
			if err != nil {
				t.Fatalf("Resolve error = %v", err)
			}
			if want := tc.path(home, wd); paths.Catalog != want {
				t.Fatalf("catalog path = %q, want %q", paths.Catalog, want)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: A profile flag wins over the profile variable
func TestAProfileFlagWinsOverTheProfileVariable(t *testing.T) {
	home := t.TempDir()
	isolatedEnv(t, home)
	profiles := filepath.Join(home, ".config", "retriever", "profiles")
	for _, name := range []string{"work", "home"} {
		if err := os.MkdirAll(filepath.Join(profiles, name), 0o755); err != nil {
			t.Fatalf("make profile %s: %v", name, err)
		}
	}
	t.Setenv("RETRIEVER_PROFILE", "home")
	in := realInputs()
	in.ProfileFlag = "work"
	paths, err := configpaths.Resolve(in)
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if paths.Profile != "work" {
		t.Fatalf("profile = %q, want work", paths.Profile)
	}
	if want := filepath.Join(profiles, "work", "sources.yml"); paths.Catalog != want {
		t.Fatalf("catalog path = %q, want %q", paths.Catalog, want)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: A profile folder that is a symbolic link is refused
func TestAProfileFolderThatIsASymbolicLinkIsRefused(t *testing.T) {
	home := t.TempDir()
	isolatedEnv(t, home)
	profiles := filepath.Join(home, ".config", "retriever", "profiles")
	target := t.TempDir()
	if err := os.MkdirAll(profiles, 0o755); err != nil {
		t.Fatalf("make the profiles folder: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(profiles, "work")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	in := realInputs()
	in.ProfileFlag = "work"
	_, err := configpaths.Resolve(in)
	wantContains(t, err, "the profile folder is a symbolic link, which is refused")
}
