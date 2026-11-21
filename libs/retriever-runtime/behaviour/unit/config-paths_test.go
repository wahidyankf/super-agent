package unit_test

import (
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
)

// profilesDir is the profiles folder under the home folder /srv/retriever/home.
const profilesDir = "/srv/retriever/home/.config/retriever/profiles"

// pathInputs returns the resolver inputs over the environment, the working directory, and the files that exist. Each
// existence check is a map lookup, so no disk is read.
func pathInputs(env map[string]string, wd string, existing ...string) configpaths.Inputs {
	present := map[string]bool{}
	for _, path := range existing {
		present[path] = true
	}
	return configpaths.Inputs{
		Getenv: lookup(env),
		Getwd:  func() (string, error) { return wd, nil },
		Exists: func(path string) bool { return present[path] },
	}
}

// home returns the environment of a run with the home folder /srv/retriever/home and no other variable.
func home() map[string]string {
	return map[string]string{"HOME": "/srv/retriever/home"}
}

// withVar returns a copy of env with one more variable.
func withVar(env map[string]string, name, value string) map[string]string {
	out := map[string]string{}
	for k, v := range env {
		out[k] = v
	}
	out[name] = value
	return out
}

// wantCode fails the test unless err has the text and the stable code given.
func wantCode(t *testing.T, err error, message, code string) {
	t.Helper()
	wantError(t, err, message)
	coded, ok := err.(interface{ ErrorCode() string })
	if !ok {
		t.Fatalf("error %#v has no stable code", err)
	}
	if coded.ErrorCode() != code {
		t.Fatalf("code = %q, want %q", coded.ErrorCode(), code)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: The first rule that applies sets the catalog path
func TestTheFirstRuleThatAppliesSetsTheCatalogPath(t *testing.T) {
	cases := []struct {
		name  string
		flag  string
		env   string
		atCwd bool
		path  string
	}{
		{"the flag wins", "/flag/sources.yml", "/env/sources.yml", true, "/flag/sources.yml"},
		{"the variable is next", "", "/env/sources.yml", true, "/env/sources.yml"},
		{"the working directory file is next", "", "", true, "/work/sources.yml"},
		{"the user folder is last", "", "", false, "/srv/retriever/home/.config/retriever/sources.yml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := home()
			if tc.env != "" {
				env = withVar(env, "RETRIEVER_CATALOG", tc.env)
			}
			var existing []string
			if tc.atCwd {
				existing = []string{"/work/sources.yml"}
			}
			in := pathInputs(env, "/work", existing...)
			in.CatalogFlag = tc.flag
			paths, err := configpaths.Resolve(in)
			if err != nil {
				t.Fatalf("Resolve error = %v", err)
			}
			if paths.Catalog != tc.path {
				t.Fatalf("catalog path = %q, want %q", paths.Catalog, tc.path)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: An empty environment variable counts as unset
func TestAnEmptyEnvironmentVariableCountsAsUnset(t *testing.T) {
	env := withVar(home(), "RETRIEVER_CATALOG", "")
	paths, err := configpaths.Resolve(pathInputs(env, "/work", "/work/sources.yml"))
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if paths.Catalog != "/work/sources.yml" {
		t.Fatalf("catalog path = %q, want the next rule", paths.Catalog)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: A relative flag is made absolute against the working directory
func TestARelativeFlagIsMadeAbsoluteAgainstTheWorkingDirectory(t *testing.T) {
	in := pathInputs(home(), "/work")
	in.CatalogFlag = "config/sources.yml"
	paths, err := configpaths.Resolve(in)
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if paths.Catalog != "/work/config/sources.yml" {
		t.Fatalf("catalog path = %q, want /work/config/sources.yml", paths.Catalog)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: The user folder is XDG_CONFIG_HOME when set, else HOME/.config
func TestTheUserFolderIsXDGConfigHomeWhenSetElseHomeDotConfig(t *testing.T) {
	cases := []struct {
		name string
		xdg  string
		path string
	}{
		{"XDG_CONFIG_HOME is set", "/xdg", "/xdg/retriever/sources.yml"},
		{"XDG_CONFIG_HOME is not set", "", "/srv/retriever/home/.config/retriever/sources.yml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := home()
			if tc.xdg != "" {
				env = withVar(env, "XDG_CONFIG_HOME", tc.xdg)
			}
			paths, err := configpaths.Resolve(pathInputs(env, "/work"))
			if err != nil {
				t.Fatalf("Resolve error = %v", err)
			}
			if paths.Catalog != tc.path {
				t.Fatalf("catalog path = %q, want %q", paths.Catalog, tc.path)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: An unreadable working directory refuses the rules that need it
func TestAnUnreadableWorkingDirectoryRefusesTheRulesThatNeedIt(t *testing.T) {
	in := configpaths.Inputs{
		Getenv: lookup(home()),
		Getwd:  func() (string, error) { return "", errors.New("permission denied") },
		Exists: func(string) bool { return false },
	}
	_, err := configpaths.Resolve(in)
	wantError(t, err, configpaths.ErrRefused.Error())
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: No home folder refuses only when the user folder is needed
func TestNoHomeFolderRefusesOnlyWhenTheUserFolderIsNeeded(t *testing.T) {
	_, err := configpaths.Resolve(pathInputs(nil, "/work"))
	wantError(t, err, configpaths.ErrRefused.Error())
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: The explicit flags resolve the paths even with no home folder
func TestTheExplicitFlagsResolveThePathsEvenWithNoHomeFolder(t *testing.T) {
	in := pathInputs(nil, "/work")
	in.CatalogFlag = "/flag/sources.yml"
	in.EnvFileFlag = "/flag/.env.local"
	paths, err := configpaths.Resolve(in)
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if paths.Catalog != "/flag/sources.yml" || paths.EnvFile != "/flag/.env.local" {
		t.Fatalf("paths = %q and %q, want the two flag paths", paths.Catalog, paths.EnvFile)
	}
}

// profileInputs returns the resolver inputs of a home folder that holds the profiles folder, with the given profile flag
// and environment. IsDir reports the profiles folder as a real folder, and IsSymlink reports the symlink given.
func profileInputs(env map[string]string, profileFlag string, symlink string) configpaths.Inputs {
	in := pathInputs(env, "/work")
	in.ProfileFlag = profileFlag
	in.IsDir = func(path string) bool { return path == profilesDir }
	in.IsSymlink = func(path string) bool { return symlink != "" && path == symlink }
	return in
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: A profile flag wins over the profile variable
func TestAProfileFlagWinsOverTheProfileVariable(t *testing.T) {
	env := withVar(home(), "RETRIEVER_PROFILE", "home")
	paths, err := configpaths.Resolve(profileInputs(env, "work", ""))
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if paths.Profile != "work" {
		t.Fatalf("profile = %q, want work", paths.Profile)
	}
	if paths.Catalog != profilesDir+"/work/sources.yml" {
		t.Fatalf("catalog path = %q, want the folder of work", paths.Catalog)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: The profile variable selects the profile when no flag is given
func TestTheProfileVariableSelectsTheProfileWhenNoFlagIsGiven(t *testing.T) {
	env := withVar(home(), "RETRIEVER_PROFILE", "home")
	paths, err := configpaths.Resolve(profileInputs(env, "", ""))
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if paths.Profile != "home" {
		t.Fatalf("profile = %q, want home", paths.Profile)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: A profiles folder with no selected profile is refused as required
func TestAProfilesFolderWithNoSelectedProfileIsRefusedAsRequired(t *testing.T) {
	_, err := configpaths.Resolve(profileInputs(home(), "", ""))
	wantCode(t, err, "no profile was selected: pass --profile <name> or set RETRIEVER_PROFILE", configpaths.ProfileRequiredCode)
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: An invalid profile name is refused with the profile invalid code
func TestAnInvalidProfileNameIsRefusedWithTheProfileInvalidCode(t *testing.T) {
	cases := []struct {
		name    string
		profile string
		message string
	}{
		{"a path", "../work", `invalid profile name "../work": use lowercase letters, digits, and single hyphens`},
		{"an upper-case name", "Work", `invalid profile name "Work": use lowercase letters, digits, and single hyphens`},
		{"a double hyphen", "team--a", `invalid profile name "team--a": use lowercase letters, digits, and single hyphens`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := configpaths.Resolve(profileInputs(home(), tc.profile, ""))
			wantCode(t, err, tc.message, configpaths.ProfileInvalidCode)
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: An explicit catalog path is refused when profiles exist
func TestAnExplicitCatalogPathIsRefusedWhenProfilesExist(t *testing.T) {
	in := profileInputs(home(), "work", "")
	in.CatalogFlag = "/flag/sources.yml"
	_, err := configpaths.Resolve(in)
	wantCode(t, err, "explicit catalog paths are refused when profiles exist: pass --profile <name>", configpaths.ProfileInvalidCode)
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: The profile flag is refused when no profiles folder exists
func TestTheProfileFlagIsRefusedWhenNoProfilesFolderExists(t *testing.T) {
	in := pathInputs(home(), "/work")
	in.ProfileFlag = "work"
	_, err := configpaths.Resolve(in)
	wantCode(t, err, "no profiles are configured: --profile needs a profiles folder", configpaths.ProfileInvalidCode)
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: A profile folder that is a symbolic link is refused
func TestAProfileFolderThatIsASymbolicLinkIsRefused(t *testing.T) {
	_, err := configpaths.Resolve(profileInputs(home(), "work", profilesDir+"/work"))
	wantCode(t, err, "the profile folder is a symbolic link, which is refused", configpaths.ProfileInvalidCode)
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: The connectors folder is the flag, else the user folder's connectors
func TestTheConnectorsFolderIsTheFlagElseTheUserFoldersConnectors(t *testing.T) {
	cases := []struct {
		name   string
		flag   string
		folder string
	}{
		{"the flag is given", "/custom/specs", "/custom/specs"},
		{"no flag is given", "", "/xdg/retriever/connectors"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := pathInputs(withVar(home(), "XDG_CONFIG_HOME", "/xdg"), "/work")
			in.ConnectorsFlag = tc.flag
			paths, err := configpaths.Resolve(in)
			if err != nil {
				t.Fatalf("Resolve error = %v", err)
			}
			if paths.Connectors != tc.folder {
				t.Fatalf("connectors folder = %q, want %q", paths.Connectors, tc.folder)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/config-paths.feature: No user configuration folder leaves no default connectors folder
func TestNoUserConfigurationFolderLeavesNoDefaultConnectorsFolder(t *testing.T) {
	in := pathInputs(nil, "/work")
	in.CatalogFlag = "/flag/sources.yml"
	in.EnvFileFlag = "/flag/env-file"
	paths, err := configpaths.Resolve(in)
	if err != nil {
		t.Fatalf("Resolve error = %v, want no error", err)
	}
	if paths.Connectors != "" {
		t.Fatalf("connectors folder = %q, want none", paths.Connectors)
	}
}
