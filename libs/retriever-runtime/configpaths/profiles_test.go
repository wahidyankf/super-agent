package configpaths

import (
	"errors"
	"strings"
	"testing"
)

// profileFixture returns Inputs whose user configuration folder holds a profiles folder with the work and personal
// profiles, so every profile rule is tested without the real disk. reads records every path the resolution checks,
// which proves what a run does and does not read.
func profileFixture(t *testing.T, env map[string]string, reads *[]string) Inputs {
	t.Helper()
	base := map[string]string{"HOME": "/srv/example/tester"}
	for k, v := range env {
		base[k] = v
	}
	record := func(kind, path string) {
		if reads != nil {
			*reads = append(*reads, kind+":"+path)
		}
	}
	return Inputs{
		Getenv: func(k string) string { return base[k] },
		Getwd: func() (string, error) {
			record("getwd", "")
			return "/work/project", nil
		},
		Exists: func(p string) bool {
			record("exists", p)
			return false
		},
		IsDir: func(p string) bool {
			record("isdir", p)
			return p == "/srv/example/tester/.config/retriever/profiles"
		},
		IsSymlink: func(p string) bool {
			record("symlink", p)
			return false
		},
	}
}

// profileCode returns the stable code of a profile error, failing the test for any other error.
func profileCode(t *testing.T, err error) string {
	t.Helper()
	var perr profileError
	if !errors.As(err, &perr) {
		t.Fatalf("error = %v, want a profile error", err)
	}
	return perr.ErrorCode()
}

// TestProfileFlagWinsOverTheVariable checks that --profile takes the personal profile when the variable names work.
func TestProfileFlagWinsOverTheVariable(t *testing.T) {
	in := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "work"}, nil)
	in.ProfileFlag = "personal"
	got, err := Resolve(in)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Profile != "personal" || got.Catalog != "/srv/example/tester/.config/retriever/profiles/personal/sources.yml" {
		t.Fatalf("Resolve = %+v, want the personal profile paths", got)
	}
	if got.EnvFile != "/srv/example/tester/.config/retriever/profiles/personal/.env.local" {
		t.Fatalf("env file = %q, want the personal env file", got.EnvFile)
	}
	if got.Connectors != "/srv/example/tester/.config/retriever/profiles/personal/connectors" {
		t.Fatalf("connectors = %q, want the personal connectors folder", got.Connectors)
	}
}

// TestProfileVariableSelectsTheProfileWhenNoFlagIsGiven checks the second selection rule.
func TestProfileVariableSelectsTheProfileWhenNoFlagIsGiven(t *testing.T) {
	got, err := Resolve(profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "work"}, nil))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Profile != "work" || !strings.HasSuffix(got.Catalog, "/profiles/work/sources.yml") {
		t.Fatalf("Resolve = %+v, want the work profile paths", got)
	}
}

// TestProfilesWithNoChoiceRefuseWithProfileRequired checks F3: profiles exist, none is chosen, and the run stops with
// the profile_required code before any catalog is read.
func TestProfilesWithNoChoiceRefuseWithProfileRequired(t *testing.T) {
	var reads []string
	_, err := Resolve(profileFixture(t, nil, &reads))
	if code := profileCode(t, err); code != "retriever.usage.profile_required" {
		t.Fatalf("code = %q, want retriever.usage.profile_required", code)
	}
	for _, r := range reads {
		if strings.HasPrefix(r, "exists:") || r == "getwd:" {
			t.Fatalf("resolution read %q, want no catalog or working-directory read", r)
		}
	}
}

// TestProfileTraversalIsRefused checks F5: a profile name that escapes its folder exits before any path is built.
func TestProfileTraversalIsRefused(t *testing.T) {
	in := profileFixture(t, nil, nil)
	in.ProfileFlag = "../other"
	_, err := Resolve(in)
	if code := profileCode(t, err); code != "retriever.usage.profile_invalid" {
		t.Fatalf("code = %q, want retriever.usage.profile_invalid", code)
	}
}

// TestInvalidProfileNamesAreRefused checks the slug rule: only lowercase letters, digits, and single hyphens.
func TestInvalidProfileNamesAreRefused(t *testing.T) {
	for _, name := range []string{"Work", "work_personal", "a--b", "-a", "a-", "a/b", "a.b", "a b", ".."} {
		in := profileFixture(t, nil, nil)
		in.ProfileFlag = name
		if _, err := Resolve(in); profileCode(t, err) != "retriever.usage.profile_invalid" {
			t.Fatalf("profile %q: error = %v, want the profile_invalid code", name, err)
		}
	}
}

// TestExplicitCatalogPathsAreRefusedWhenProfilesExist checks F2: an explicit path bypasses the profile, so it is a
// usage error, whether it comes from a flag or from an environment variable.
func TestExplicitCatalogPathsAreRefusedWhenProfilesExist(t *testing.T) {
	flags := []Inputs{}
	cat := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal"}, nil)
	cat.CatalogFlag = "/elsewhere/sources.yml"
	flags = append(flags, cat)
	envf := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal"}, nil)
	envf.EnvFileFlag = "/elsewhere/.env"
	flags = append(flags, envf)
	conn := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal"}, nil)
	conn.ConnectorsFlag = "/elsewhere/specs"
	flags = append(flags, conn)
	catVar := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal", "RETRIEVER_CATALOG": "/elsewhere/sources.yml"}, nil)
	flags = append(flags, catVar)
	envVar := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal", "RETRIEVER_ENV_FILE": "/elsewhere/.env"}, nil)
	flags = append(flags, envVar)
	for i, in := range flags {
		if _, err := Resolve(in); profileCode(t, err) != "retriever.usage.profile_invalid" {
			t.Fatalf("case %d: error = %v, want the profile_invalid code", i, err)
		}
	}
}

// TestProfileWithNoProfilesFolderIsRefused checks that --profile without a profiles folder is a usage error, so the
// run never falls back to the single catalog silently.
func TestProfileWithNoProfilesFolderIsRefused(t *testing.T) {
	in := fixture()
	in.ProfileFlag = "personal"
	if _, err := Resolve(in); profileCode(t, err) != "retriever.usage.profile_invalid" {
		t.Fatalf("error = %v, want the profile_invalid code", err)
	}
	in = fixture()
	in.Getenv = func(k string) string {
		if k == "RETRIEVER_PROFILE" {
			return "personal"
		}
		return map[string]string{"HOME": "/srv/example/tester"}[k]
	}
	if _, err := Resolve(in); profileCode(t, err) != "retriever.usage.profile_invalid" {
		t.Fatalf("variable: error = %v, want the profile_invalid code", err)
	}
}

// TestSymlinkedProfileFolderIsRefused checks F5: a profile folder that is a symbolic link is refused, and the message
// names no path.
func TestSymlinkedProfileFolderIsRefused(t *testing.T) {
	in := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal"}, nil)
	in.IsSymlink = func(p string) bool { return strings.HasSuffix(p, "/profiles/personal") }
	_, err := Resolve(in)
	if code := profileCode(t, err); code != "retriever.usage.profile_invalid" {
		t.Fatalf("code = %q, want retriever.usage.profile_invalid", code)
	}
	if strings.Contains(err.Error(), "/srv/") || strings.Contains(err.Error(), ".config") {
		t.Fatalf("error names a path: %q", err)
	}
}

// TestProfilesFolderWithoutAHomeIsRefused checks that a folder check without a user configuration folder is still a
// refusal: the environment refused before any profile was selected.
func TestProfilesFolderWithoutAHomeIsRefused(t *testing.T) {
	in := Inputs{
		Getenv:    func(string) string { return "" },
		Getwd:     func() (string, error) { return "/work/project", nil },
		Exists:    func(string) bool { return false },
		IsDir:     func(string) bool { return false },
		IsSymlink: func(string) bool { return false },
	}
	if _, err := Resolve(in); !errors.Is(err, ErrRefused) {
		t.Fatalf("error = %v, want ErrRefused", err)
	}
}

// TestExplicitFlagsWinWithoutAHome checks the refusal contract: a missing home does not matter when explicit flags
// name every path, because an unnameable user folder means no profiles folder to consider.
func TestExplicitFlagsWinWithoutAHome(t *testing.T) {
	in := Inputs{
		CatalogFlag: "/flag/sources.yml",
		EnvFileFlag: "/flag/.env",
		Getenv:      func(string) string { return "" },
		Getwd:       func() (string, error) { return "/work/project", nil },
		Exists:      func(string) bool { return false },
		IsDir:       func(string) bool { return false },
	}
	got, err := Resolve(in)
	if err != nil || got.Catalog != "/flag/sources.yml" || got.EnvFile != "/flag/.env" {
		t.Fatalf("Resolve = %+v, %v; want the flag paths with no home", got, err)
	}
}

// TestProfileResolutionWithoutASymlinkCheckSkipsTheCheck checks that a nil symlink check resolves the profile without
// one: the check runs only when the caller supplies it.
func TestProfileResolutionWithoutASymlinkCheckSkipsTheCheck(t *testing.T) {
	in := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal"}, nil)
	in.IsSymlink = nil
	got, err := Resolve(in)
	if err != nil || got.Profile != "personal" {
		t.Fatalf("Resolve = %+v, %v; want the personal profile", got, err)
	}
}

// TestPersonalRunReadsNoWorkFolder checks F1 at the resolution level: resolving the personal profile never checks the
// work folder. The recording file system of the loader proves the same for the catalog read.
func TestPersonalRunReadsNoWorkFolder(t *testing.T) {
	var reads []string
	in := profileFixture(t, map[string]string{"RETRIEVER_PROFILE": "personal"}, &reads)
	if _, err := Resolve(in); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, r := range reads {
		if strings.Contains(r, "/profiles/work") {
			t.Fatalf("personal resolution touched the work folder: %q in %q", r, reads)
		}
	}
}
