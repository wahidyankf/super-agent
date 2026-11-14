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

	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
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

// TempFile is the part of a temporary export file that a write uses. *os.File satisfies it. The interface lets a test
// make one step fail, which a real folder cannot do on demand.
type TempFile interface {
	io.Writer
	Name() string
	Chmod(mode os.FileMode) error
	Close() error
}

// CreateTemp makes the temporary file of one write inside dir. The composition root passes a function over os.CreateTemp.
type CreateTemp func(dir, pattern string) (TempFile, error)

// Folder holds the folder calls a write makes. The composition root passes the calls of package os.
type Folder struct {
	MkdirAll func(path string, perm os.FileMode) error
	Rename   func(oldpath, newpath string) error
	Remove   func(name string) error
}

// Sink implements ports.ExportSink over one folder. Build it with New.
type Sink struct {
	dir    string
	create CreateTemp
	fsys   Folder
}

var _ ports.ExportSink = (*Sink)(nil)

// New returns a sink that writes into dir, through the temporary file and folder calls it is given. The folder is created
// on the first write.
func New(dir string, create CreateTemp, folder Folder) *Sink {
	return &Sink{dir: dir, create: create, fsys: folder}
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
	if err := s.fsys.MkdirAll(s.dir, 0o755); err != nil {
		return "", fmt.Errorf("creating the export folder: %w", err)
	}
	target := filepath.Join(s.dir, name)
	if err := s.writeAtomic(s.dir, target, data); err != nil {
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
func (s *Sink) writeAtomic(dir, target string, data []byte) (err error) {
	tmp, err := s.create(dir, ".export-*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = s.fsys.Remove(tmpName)
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
	if err = s.fsys.Rename(tmpName, target); err != nil {
		return fmt.Errorf("placing the export: %w", err)
	}
	return nil
}
