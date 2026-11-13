// Package markdownfiles is the default export sink. It writes each exported document as one Markdown file in the folder
// that the --out flag names. A write replaces an existing file of the same name, through a temporary file in the same
// folder and a rename, so a reader never sees a half-written file.
package markdownfiles

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// Errors that a write returns before it touches the disk.
var (
	// ErrNoFolder reports a sink that was given no folder.
	ErrNoFolder = errors.New("no export folder was given")
	// ErrBadName reports a file name that is not one plain Markdown file name.
	ErrBadName = errors.New("the export file name must be one plain name ending in .md")
)

// fileMode is the permission of an exported file: readable by everyone, writable by its owner.
const fileMode os.FileMode = 0o644

// tempFile is the part of a temporary export file that a write uses. *os.File satisfies it. The interface lets a test
// make one step fail, which a real folder cannot do on demand.
type tempFile interface {
	io.Writer
	Name() string
	Chmod(mode os.FileMode) error
	Close() error
}

// createTempFunc makes the temporary file of one write inside dir. New uses os.CreateTemp.
type createTempFunc func(dir, pattern string) (tempFile, error)

// osCreateTemp is the real creator. It returns a nil interface, not a typed nil, when the file cannot be made.
func osCreateTemp(dir, pattern string) (tempFile, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Sink implements ports.ExportSink over one folder. Build it with New.
type Sink struct {
	dir    string
	create createTempFunc
}

var _ ports.ExportSink = (*Sink)(nil)

// New returns a sink that writes into dir. The folder is created on the first write.
func New(dir string) *Sink {
	return &Sink{dir: dir, create: osCreateTemp}
}

// Write stores data as the file name inside the sink's folder and returns the path it wrote. The name is checked before
// the folder is created, so a refused name leaves the disk as it was.
func (s *Sink) Write(_ context.Context, name string, data []byte) (string, error) {
	if s.dir == "" {
		return "", ErrNoFolder
	}
	if err := checkName(name); err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", fmt.Errorf("creating the export folder: %w", err)
	}
	target := filepath.Join(s.dir, name)
	if err := writeAtomic(s.create, s.dir, target, data); err != nil {
		return "", err
	}
	return target, nil
}

// checkName accepts one file name: no separator, no NUL byte, no dot-only name, and the .md extension with a stem.
func checkName(name string) error {
	switch {
	case name == "", name == ".", name == "..":
		return ErrBadName
	case strings.ContainsAny(name, "/\\\x00"):
		return ErrBadName
	case !strings.HasSuffix(name, ".md") || len(name) <= len(".md"):
		return ErrBadName
	}
	return nil
}

// writeAtomic writes data to a temporary file in dir and renames it to target. The temporary file is removed when a
// step fails.
func writeAtomic(create createTempFunc, dir, target string, data []byte) (err error) {
	tmp, err := create(dir, ".export-*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return errors.Join(fmt.Errorf("writing the export: %w", err), tmp.Close())
	}
	if err = tmp.Chmod(fileMode); err != nil {
		return errors.Join(fmt.Errorf("setting the export mode: %w", err), tmp.Close())
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("closing the export: %w", err)
	}
	if err = os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("placing the export: %w", err)
	}
	return nil
}
