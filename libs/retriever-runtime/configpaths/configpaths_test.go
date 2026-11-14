package configpaths

import (
	"errors"
	"path/filepath"
	"testing"
)

// fixture returns Inputs whose working directory, home folder, and files are fixed, so every rule is tested without the
// real environment.
func fixture(files ...string) Inputs {
	present := map[string]bool{}
	for _, f := range files {
		present[f] = true
	}
	env := map[string]string{"HOME": "/srv/example/tester"}
	return Inputs{
		Getenv: func(k string) string { return env[k] },
		Getwd:  func() (string, error) { return "/work/project", nil },
		Exists: func(p string) bool { return present[p] },
	}
}

// TestCatalogFlagWinsOverEverything checks the first rule: an explicit --catalog path is used, even when the variable and
// the working-directory file are both set.
func TestCatalogFlagWinsOverEverything(t *testing.T) {
	in := fixture("/work/project/sources.yml")
	in.CatalogFlag = "/etc/retriever/catalog.yml"
	in.Getenv = func(k string) string {
		if k == "RETRIEVER_CATALOG" {
			return "/from/env.yml"
		}
		return map[string]string{"HOME": "/srv/example/tester"}[k]
	}
	got, err := Resolve(in)
	if err != nil || got.Catalog != "/etc/retriever/catalog.yml" {
		t.Fatalf("Resolve = %q, %v; want the flag path", got.Catalog, err)
	}
}

// TestCatalogEnvironmentWinsOverTheDefaults checks the second rule: RETRIEVER_CATALOG is used when no flag is given.
func TestCatalogEnvironmentWinsOverTheDefaults(t *testing.T) {
	in := fixture("/work/project/sources.yml")
	in.Getenv = func(k string) string {
		if k == "RETRIEVER_CATALOG" {
			return "/from/env.yml"
		}
		return map[string]string{"HOME": "/srv/example/tester"}[k]
	}
	got, err := Resolve(in)
	if err != nil || got.Catalog != "/from/env.yml" {
		t.Fatalf("Resolve = %q, %v; want the environment path", got.Catalog, err)
	}
}

// TestEmptyEnvironmentValueIsUnset checks the presence rule: an exported empty variable is not an override.
func TestEmptyEnvironmentValueIsUnset(t *testing.T) {
	in := fixture("/work/project/sources.yml")
	in.Getenv = func(k string) string {
		if k == "RETRIEVER_CATALOG" {
			return ""
		}
		return map[string]string{"HOME": "/srv/example/tester"}[k]
	}
	got, err := Resolve(in)
	if err != nil || got.Catalog != "/work/project/sources.yml" {
		t.Fatalf("Resolve = %q, %v; want the working-directory default", got.Catalog, err)
	}
}

// TestCatalogDefaultsToTheWorkingDirectoryWhenThatFileExists checks the third rule.
func TestCatalogDefaultsToTheWorkingDirectoryWhenThatFileExists(t *testing.T) {
	got, err := Resolve(fixture("/work/project/sources.yml"))
	if err != nil || got.Catalog != "/work/project/sources.yml" {
		t.Fatalf("Resolve = %q, %v; want ./sources.yml", got.Catalog, err)
	}
}

// TestCatalogFallsBackToTheUserConfigFolder checks the last rule: with no flag, variable, or working-directory file, the
// catalog is read from the user configuration folder, which honours XDG_CONFIG_HOME.
func TestCatalogFallsBackToTheUserConfigFolder(t *testing.T) {
	got, err := Resolve(fixture())
	if err != nil || got.Catalog != "/srv/example/tester/.config/retriever/sources.yml" {
		t.Fatalf("Resolve = %q, %v; want ~/.config/retriever/sources.yml", got.Catalog, err)
	}
	in := fixture()
	in.Getenv = func(k string) string {
		return map[string]string{"HOME": "/srv/example/tester", "XDG_CONFIG_HOME": "/xdg"}[k]
	}
	got, err = Resolve(in)
	if err != nil || got.Catalog != "/xdg/retriever/sources.yml" {
		t.Fatalf("Resolve with XDG = %q, %v; want /xdg/retriever/sources.yml", got.Catalog, err)
	}
}

// TestEmptyXDGConfigHomeFallsBackToHome checks that an exported empty XDG_CONFIG_HOME is not a base folder.
func TestEmptyXDGConfigHomeFallsBackToHome(t *testing.T) {
	in := fixture()
	in.Getenv = func(k string) string {
		return map[string]string{"HOME": "/srv/example/tester", "XDG_CONFIG_HOME": ""}[k]
	}
	got, err := Resolve(in)
	if err != nil || got.Catalog != "/srv/example/tester/.config/retriever/sources.yml" {
		t.Fatalf("Resolve = %q, %v; want the HOME-based path", got.Catalog, err)
	}
}

// TestRelativeFlagIsResolvedAgainstTheWorkingDirectory checks that a relative path becomes absolute from the working
// directory, so the loaders never depend on a later change of directory.
func TestRelativeFlagIsResolvedAgainstTheWorkingDirectory(t *testing.T) {
	in := fixture()
	in.CatalogFlag = "catalogs/work.yml"
	in.EnvFileFlag = "secrets/.env.work"
	got, err := Resolve(in)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if want := filepath.Join("/work/project", "catalogs/work.yml"); got.Catalog != want {
		t.Fatalf("catalog = %q; want %q", got.Catalog, want)
	}
	if want := filepath.Join("/work/project", "secrets/.env.work"); got.EnvFile != want {
		t.Fatalf("env file = %q; want %q", got.EnvFile, want)
	}
}

// TestEnvFileRulesMirrorTheCatalogRules checks the env-file path: the flag, then the variable, then ./.env.local when it
// exists, then the user configuration folder.
func TestEnvFileRulesMirrorTheCatalogRules(t *testing.T) {
	in := fixture("/work/project/.env.local")
	in.EnvFileFlag = "/flag/.env"
	if got, _ := Resolve(in); got.EnvFile != "/flag/.env" {
		t.Fatalf("flag: env file = %q", got.EnvFile)
	}
	in = fixture("/work/project/.env.local")
	in.Getenv = func(k string) string {
		return map[string]string{"HOME": "/srv/example/tester", "RETRIEVER_ENV_FILE": "/env/.env"}[k]
	}
	if got, _ := Resolve(in); got.EnvFile != "/env/.env" {
		t.Fatalf("variable: env file = %q", got.EnvFile)
	}
	if got, _ := Resolve(fixture("/work/project/.env.local")); got.EnvFile != "/work/project/.env.local" {
		t.Fatalf("working directory: env file = %q", got.EnvFile)
	}
	if got, _ := Resolve(fixture()); got.EnvFile != "/srv/example/tester/.config/retriever/.env.local" {
		t.Fatalf("user config: env file = %q", got.EnvFile)
	}
}

// TestUnknownWorkingDirectoryIsARefusal checks that a working directory that cannot be read stops the resolution with
// the refusal error, which the CLI maps to the environment status.
func TestUnknownWorkingDirectoryIsARefusal(t *testing.T) {
	in := fixture()
	in.Getwd = func() (string, error) { return "", errors.New("no directory") }
	_, err := Resolve(in)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("error = %v; want ErrRefused", err)
	}
}

// TestMissingHomeIsARefusalOnlyWhenTheUserFolderIsNeeded checks that a missing HOME does not matter when a path is given
// explicitly, and refuses when the user configuration folder is the only source left.
func TestMissingHomeIsARefusalOnlyWhenTheUserFolderIsNeeded(t *testing.T) {
	noHome := func(in Inputs) Inputs {
		in.Getenv = func(string) string { return "" }
		return in
	}
	in := noHome(fixture("/work/project/sources.yml", "/work/project/.env.local"))
	if _, err := Resolve(in); err != nil {
		t.Fatalf("Resolve with working-directory files and no HOME: %v", err)
	}
	in = noHome(fixture())
	if _, err := Resolve(in); !errors.Is(err, ErrRefused) {
		t.Fatalf("Resolve with no files and no HOME: error = %v; want ErrRefused", err)
	}
}
