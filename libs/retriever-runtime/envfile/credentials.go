// Package envfile supplies source API keys and identities from the process environment or from one local env file.
//
// The process environment wins. When it has no non-empty value for a key, the env file is read, and only the one
// variable is taken from it. The file is read at most once per call, and it is opened only after a check that it is a
// regular file and not a symlink. A parse error names the line number and never its text, because a line may hold a
// secret.
package envfile

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

const (
	// envFileName is the local env file that the repository root holds.
	envFileName = ".env.local"
)

// Files are the file calls a read makes. FS holds the env file, which the reader names by the base of its path, so FS is
// the folder that holds it. The composition root passes os.DirFS of that folder; a unit test passes an in-memory fs.FS.
// SameFile reports whether two descriptions are the same file.
type Files struct {
	FS       fs.FS
	SameFile func(a, b fs.FileInfo) bool
}

// Credentials implements ports.Credentials and ports.Identities. Build it with NewWithFiles.
type Credentials struct {
	envPath string
	getenv  func(string) string
	files   Files
}

var (
	_ ports.Credentials = (*Credentials)(nil)
	_ ports.Identities  = (*Credentials)(nil)
)

// NewWithFiles returns Credentials that read the environment through getenv and the env file at envPath through files.
// envPath must be absolute. A missing file is not an error: the key is then reported as missing. A unit test passes a
// fake lookup and an in-memory set, so no real variable or file is read.
func NewWithFiles(envPath string, getenv func(string) string, files Files) (*Credentials, error) {
	if !filepath.IsAbs(envPath) {
		return nil, errors.New("the env file path must be absolute")
	}
	return &Credentials{envPath: envPath, getenv: getenv, files: files}, nil
}

// APIKey returns the key that ref names. The process value wins over the file value. An empty value in either place counts
// as missing, and a missing key returns a domain.MissingKeyError that names the variable and never a value. The zero
// reference is refused, so the reader never reads a variable that no reference names.
func (c *Credentials) APIKey(ref domain.CredentialRef) (domain.ApiKey, error) {
	variable := ref.Name()
	if variable == "" {
		return domain.ApiKey{}, domain.ErrInvalidCredentialRef
	}
	value, err := c.value(variable)
	if err != nil {
		return domain.ApiKey{}, err
	}
	return domain.NewApiKey(variable, value)
}

// Identity returns the e-mail address that ref names, with the same lookup order as APIKey: the process value first, then
// the one variable from the local env file. A missing address returns a domain.MissingKeyError, and a malformed one returns
// a domain.InvalidIdentityError; neither carries a value. The zero reference is refused.
func (c *Credentials) Identity(ref domain.IdentityRef) (domain.Identity, error) {
	variable := ref.Name()
	if variable == "" {
		return domain.Identity{}, domain.ErrInvalidIdentityRef
	}
	value, err := c.value(variable)
	if err != nil {
		return domain.Identity{}, err
	}
	return domain.NewIdentity(variable, value)
}

// value returns the value of one variable: the process value when it is non-empty, else the value in the local env file.
// An empty value in both places is returned as "" with no error, so the caller reports it as missing.
func (c *Credentials) value(variable string) (string, error) {
	if value := c.getenv(variable); value != "" {
		return value, nil
	}
	return c.fileValue(variable)
}

// fileValue returns the value of variable in the local env file. It returns "" with no error when the file does not
// exist. It refuses a path that is not a regular file, including a symlink, and a file that changes while it is opened.
func (c *Credentials) fileValue(variable string) (value string, err error) {
	name := filepath.Base(c.envPath)
	info, err := fs.Lstat(c.files.FS, name)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("checking local env file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("local env file must be a regular file, not a symlink or a folder")
	}
	f, err := c.files.FS.Open(name)
	if err != nil {
		return "", fmt.Errorf("opening local env file: %w", err)
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()
	opened, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("checking local env file: %w", err)
	}
	if !c.files.SameFile(info, opened) {
		return "", errors.New("local env file changed while it was being opened")
	}
	return lookup(f, variable)
}
