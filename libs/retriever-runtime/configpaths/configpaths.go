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
package configpaths

import (
	"errors"
	"path/filepath"
)

// ErrRefused reports that the environment cannot name a path the rules need, such as a working directory that cannot be
// read, or no home folder when the user configuration folder is the only source left. The CLI maps it to the
// environment-refused exit status.
var ErrRefused = errors.New("the environment cannot name a configuration path")

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
}

// Paths are the absolute paths the loaders read.
type Paths struct {
	Catalog string
	EnvFile string
	// Connectors is the folder of connector spec files, or "" when no folder applies. The default folder is empty only
	// when the environment names no user configuration folder.
	Connectors string
}

// Names of the files and of the environment variables that override them.
const (
	catalogName    = "sources.yml"
	envFileName    = ".env.local"
	catalogVar     = "RETRIEVER_CATALOG"
	envFileVar     = "RETRIEVER_ENV_FILE"
	userFolderName = "retriever"
	connectorsName = "connectors"
)

// Resolve applies the rules to both files and to the connectors folder. It reads the working directory only when a rule
// needs it, and the user configuration folder only when no earlier rule applies.
func Resolve(in Inputs) (Paths, error) {
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
