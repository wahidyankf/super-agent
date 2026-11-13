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
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

const (
	// envFileName is the local env file that the repository root holds.
	envFileName = ".env.local"
)

// fileReader is the part of an open env file that a read uses. *os.File satisfies it.
type fileReader interface {
	io.Reader
	Stat() (fs.FileInfo, error)
	Close() error
}

// fileOps are the file calls a read makes. New uses the real files; a test replaces them to make one step fail.
type fileOps struct {
	lstat func(name string) (fs.FileInfo, error)
	open  func(name string) (fileReader, error)
}

// realFiles is the set of file calls New uses.
var realFiles = fileOps{lstat: os.Lstat, open: openFile}

// openFile opens an env file. It returns a nil interface, not a typed nil, when the open fails.
func openFile(name string) (fileReader, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Credentials implements ports.Credentials and ports.Identities. Build it with New.
type Credentials struct {
	envPath string
	getenv  func(string) string
	files   fileOps
}

var (
	_ ports.Credentials = (*Credentials)(nil)
	_ ports.Identities  = (*Credentials)(nil)
)

// New returns Credentials that read the local env file at envPath, which the configpaths rules choose. envPath must be
// absolute. A missing file is not an error: the key is then reported as missing.
func New(envPath string) (*Credentials, error) {
	if !filepath.IsAbs(envPath) {
		return nil, errors.New("the env file path must be absolute")
	}
	return &Credentials{envPath: envPath, getenv: os.Getenv, files: realFiles}, nil
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
	info, err := c.files.lstat(c.envPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("checking local env file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("local env file must be a regular file, not a symlink or a folder")
	}
	f, err := c.files.open(c.envPath)
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
	if !os.SameFile(info, opened) {
		return "", errors.New("local env file changed while it was being opened")
	}
	return lookup(f, variable)
}
