// Package configpaths decides where the sources catalog and the local env file are read from. The rules are published in
// the command reference, and this package is their one implementation.
//
// For each file the first rule that applies wins:
//
//  1. the explicit flag (--catalog, --env-file);
//  2. the environment variable (RETRIEVER_CATALOG, RETRIEVER_ENV_FILE), when it is set and not empty;
//  3. the file of the same name in the working directory (./sources.yml, ./.env.local), when it exists;
//  4. the user configuration folder, $XDG_CONFIG_HOME/retriever or ~/.config/retriever, when XDG_CONFIG_HOME is unset or
//     empty.
//
// A relative path from the flag or the variable is made absolute against the working directory. The package reads no
// file itself: the caller passes in the environment, the working directory, and an existence check.
//
// When the user configuration folder holds a profiles folder, the four rules above do not apply: the catalog, the env
// file, and the connectors folder come from the folder of the profile the --profile flag or the RETRIEVER_PROFILE
// variable names, and an explicit path is a usage error.
package configpaths

import (
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ErrRefused reports that the environment cannot name a path the rules need, such as a working directory that cannot be
// read, or no home folder when the user configuration folder is the only source left. The CLI maps it to the
// environment-refused exit status.
var ErrRefused = errors.New("the environment cannot name a configuration path")

// Stable codes of the profile failures. Every profile failure exits 2, never 125: the environment named the paths, and
// the selection is what broke.
const (
	// ProfileRequiredCode is the code of a run with a profiles folder and no selected profile.
	ProfileRequiredCode = "retriever.usage.profile_required"
	// ProfileInvalidCode is the code of a profile selection the run refuses.
	ProfileInvalidCode = "retriever.usage.profile_invalid"
)

// profileError reports a profile selection the run refuses. Its text is fixed, except for an invalid name, which quotes
// the name the caller typed: a name is never a value. Code returns the stable code of the failure.
type profileError struct {
	code string
	rule string
}

// Error returns the rule that broke.
func (e profileError) Error() string { return e.rule }

// ErrorCode returns the stable code of the failure.
func (e profileError) ErrorCode() string { return e.code }

// Inputs are the values the rules read. The caller fills each one from the real process, and a test fills them from a
// fixture.
type Inputs struct {
	// CatalogFlag and EnvFileFlag hold the values of --catalog and --env-file, or "" when the flag is absent.
	CatalogFlag string
	EnvFileFlag string
	// Getenv reads one environment variable. An unset variable and an empty one both read as "".
	Getenv func(string) string
	// Getwd returns the working directory.
	Getwd func() (string, error)
	// Exists reports whether a regular file exists at an absolute path.
	Exists func(path string) bool
	// ConnectorsFlag holds the value of --connectors-dir, or "" when the flag is absent.
	ConnectorsFlag string
	// ProfileFlag holds the value of --profile, or "" when the flag is absent.
	ProfileFlag string
	// IsDir reports whether a path is a real folder, following no symbolic link. A nil check reports no folder, so a
	// caller without a folder check resolves the single catalog as before.
	IsDir func(path string) bool
	// IsSymlink reports whether a path is a symbolic link. A nil check reports none.
	IsSymlink func(path string) bool
}

// Paths are the absolute paths the loaders read.
type Paths struct {
	Catalog string
	EnvFile string
	// Connectors is the folder of connector spec files, or "" when no folder applies. The default folder is empty only
	// when the environment names no user configuration folder.
	Connectors string
	// Profile is the active profile name, or "" when the run has no profiles folder and reads the single catalog.
	Profile string
}

// Names of the files, of the folders, and of the environment variables that override them.
const (
	catalogName    = "sources.yml"
	envFileName    = ".env.local"
	catalogVar     = "RETRIEVER_CATALOG"
	envFileVar     = "RETRIEVER_ENV_FILE"
	profileVar     = "RETRIEVER_PROFILE"
	userFolderName = "retriever"
	connectorsName = "connectors"
	profilesName   = "profiles"
)

// Resolve applies the rules to both files and to the connectors folder. It reads the working directory only when a rule
// needs it, and the user configuration folder only when no earlier rule applies.
//
// When the user configuration folder holds a profiles folder, the run has one active profile and reads only its folder:
// the catalog, the env file, and the connectors folder of the profile the --profile flag or the RETRIEVER_PROFILE
// variable names. An explicit path flag or variable is then a usage error, and a run with no choice fails with the
// profile_required code before any file is read.
func Resolve(in Inputs) (Paths, error) {
	profilesDir, present, err := profilesFolder(in)
	if err != nil {
		return Paths{}, err
	}
	if present {
		return resolveProfiled(in, profilesDir)
	}
	if in.ProfileFlag != "" || in.Getenv(profileVar) != "" {
		return Paths{}, profileError{code: ProfileInvalidCode,
			rule: "no profiles are configured: --profile needs a profiles folder"}
	}
	catalog, err := resolveOne(in, in.CatalogFlag, catalogVar, catalogName)
	if err != nil {
		return Paths{}, err
	}
	envFile, err := resolveOne(in, in.EnvFileFlag, envFileVar, envFileName)
	if err != nil {
		return Paths{}, err
	}
	connectors, err := resolveConnectors(in)
	if err != nil {
		return Paths{}, err
	}
	return Paths{Catalog: catalog, EnvFile: envFile, Connectors: connectors}, nil
}

// profilesFolder returns the profiles folder of the user configuration folder and whether it is a real folder. A nil
// folder check means the caller reads the single catalog, so no folder exists. When the environment names no user
// configuration folder, there is no profiles folder to consider: the legacy rules below refuse only if they need the
// folder. The working directory is never read.
func profilesFolder(in Inputs) (dir string, present bool, err error) {
	if in.IsDir == nil {
		return "", false, nil
	}
	base, err := userConfigBase(in)
	if err != nil {
		return "", false, nil
	}
	dir = filepath.Join(base, userFolderName, profilesName)
	return dir, in.IsDir(dir), nil
}

// resolveProfiled resolves the three paths of the active profile inside profilesDir. The flag wins over the variable.
// The resolution checks only the profiles folder and the profile folder: it reads no catalog file, no working
// directory, and no other profile's folder.
func resolveProfiled(in Inputs, profilesDir string) (Paths, error) {
	if in.CatalogFlag != "" || in.EnvFileFlag != "" || in.ConnectorsFlag != "" ||
		in.Getenv(catalogVar) != "" || in.Getenv(envFileVar) != "" {
		return Paths{}, profileError{code: ProfileInvalidCode,
			rule: "explicit catalog paths are refused when profiles exist: pass --profile <name>"}
	}
	name := in.ProfileFlag
	if name == "" {
		name = in.Getenv(profileVar)
	}
	if name == "" {
		return Paths{}, profileError{code: ProfileRequiredCode,
			rule: "no profile was selected: pass --profile <name> or set RETRIEVER_PROFILE"}
	}
	if !validProfileName(name) {
		return Paths{}, profileError{code: ProfileInvalidCode,
			rule: "invalid profile name " + strconv.Quote(name) + ": use lowercase letters, digits, and single hyphens"}
	}
	profileDir := filepath.Join(profilesDir, name)
	if in.IsSymlink != nil && in.IsSymlink(profileDir) {
		return Paths{}, profileError{code: ProfileInvalidCode,
			rule: "the profile folder is a symbolic link, which is refused"}
	}
	return Paths{
		Catalog:    filepath.Join(profileDir, catalogName),
		EnvFile:    filepath.Join(profileDir, envFileName),
		Connectors: filepath.Join(profileDir, connectorsName),
		Profile:    name,
	}, nil
}

// validProfileName reports whether name is a lowercase slug of letters, digits, and single hyphens: it starts and ends
// with a letter or digit, and holds no path separator, so it stays inside the profiles folder.
func validProfileName(name string) bool {
	return profileSlug.MatchString(name)
}

// profileSlug is the shape of a profile name.
var profileSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ProfileEnvPrefix returns the required prefix of a credential or identity name of the profile: the profile name in
// upper case, with hyphens as underscores, followed by an underscore. The profile work uses WORK_.
func ProfileEnvPrefix(profile string) string {
	return strings.ToUpper(strings.ReplaceAll(profile, "-", "_")) + "_"
}

// resolveConnectors returns the folder of connector specs: the flag when it is given, else the connectors folder of the
// user configuration folder. The default is returned whether or not the folder exists, and the loader treats a missing
// default as no specs. When no user configuration folder can be named, the result is "" with no error. The rule has no
// environment variable of its own.
func resolveConnectors(in Inputs) (string, error) {
	if in.ConnectorsFlag != "" {
		return absolute(in, in.ConnectorsFlag)
	}
	base, err := userConfigBase(in)
	if err != nil {
		return "", nil
	}
	return filepath.Join(base, userFolderName, connectorsName), nil
}

// resolveOne applies the four rules to one file.
func resolveOne(in Inputs, flag, variable, name string) (string, error) {
	if flag != "" {
		return absolute(in, flag)
	}
	if value := in.Getenv(variable); value != "" {
		return absolute(in, value)
	}
	wd, err := in.Getwd()
	if err != nil || wd == "" {
		return "", ErrRefused
	}
	local := filepath.Join(wd, name)
	if in.Exists(local) {
		return local, nil
	}
	base, err := userConfigBase(in)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, userFolderName, name), nil
}

// absolute returns path as an absolute path, resolving a relative one against the working directory.
func absolute(in Inputs, path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	wd, err := in.Getwd()
	if err != nil || wd == "" {
		return "", ErrRefused
	}
	return filepath.Join(wd, path), nil
}

// userConfigBase returns XDG_CONFIG_HOME when it is set and not empty, else the .config folder under HOME.
func userConfigBase(in Inputs) (string, error) {
	if xdg := in.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}
	home := in.Getenv("HOME")
	if home == "" {
		return "", ErrRefused
	}
	return filepath.Join(home, ".config"), nil
}
