// Package markdownfiles is the default export sink. It writes each exported document as one Markdown file in the folder
// that the --out flag names. A write replaces an existing file of the same name, through a temporary file in the same
// folder and a rename, so a reader never sees a half-written file.
package markdownfiles

import (
	"context"
	"errors"
	"fmt"
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

// Sink implements ports.ExportSink over one folder. Build it with New.
type Sink struct {
	dir string
}

var _ ports.ExportSink = (*Sink)(nil)

// New returns a sink that writes into dir. The folder is created on the first write.
func New(dir string) *Sink {
	return &Sink{dir: dir}
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
	if err := writeAtomic(s.dir, target, data); err != nil {
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
func writeAtomic(dir, target string, data []byte) (err error) {
	tmp, err := os.CreateTemp(dir, ".export-*.tmp")
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
