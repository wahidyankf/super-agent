package unit_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-runtime/markdownfiles"
)

// memDisk is an in-memory export folder. It holds the files by their full path and the folders it was asked to make. Its
// calls are the ones the sink makes: a temporary file, a write, a mode change, a close, a rename, a remove, and a folder
// make. Each call can be made to fail through failStep, which a real folder cannot do on demand.
type memDisk struct {
	files    map[string]string
	dirs     map[string]bool
	failStep string
	counter  int
}

// newMemDisk returns an empty in-memory folder.
func newMemDisk() *memDisk {
	return &memDisk{files: map[string]string{}, dirs: map[string]bool{}}
}

// stepError is the failure of one named step.
func stepError(step string) error { return errors.New(step + " failed") }

// create makes a temporary file in dir, as os.CreateTemp does.
func (d *memDisk) create(dir, _ string) (markdownfiles.TempFile, error) {
	if d.failStep == "create" {
		return nil, stepError("create")
	}
	d.counter++
	name := dir + "/.export-" + string(rune('a'+d.counter)) + ".tmp"
	d.files[name] = ""
	return &memTemp{disk: d, name: name}, nil
}

// mkdirAll makes the folder and its parents, as os.MkdirAll does.
func (d *memDisk) mkdirAll(path string, _ fs.FileMode) error {
	if d.failStep == "mkdir" {
		return stepError("mkdir")
	}
	d.dirs[path] = true
	return nil
}

// rename moves a file, as os.Rename does.
func (d *memDisk) rename(oldpath, newpath string) error {
	if d.failStep == "rename" {
		return stepError("rename")
	}
	d.files[newpath] = d.files[oldpath]
	delete(d.files, oldpath)
	return nil
}

// remove deletes a file, as os.Remove does.
func (d *memDisk) remove(name string) error {
	delete(d.files, name)
	return nil
}

// memTemp is a temporary file of memDisk. Its writes accumulate in the folder under its own name.
type memTemp struct {
	disk *memDisk
	name string
}

// Write appends the bytes to the file, or fails at the write step.
func (m *memTemp) Write(p []byte) (int, error) {
	if m.disk.failStep == "write" {
		return 0, stepError("write")
	}
	m.disk.files[m.name] += string(p)
	return len(p), nil
}

// Name returns the path of the temporary file.
func (m *memTemp) Name() string { return m.name }

// Chmod changes nothing, or fails at the mode step.
func (m *memTemp) Chmod(fs.FileMode) error {
	if m.disk.failStep == "chmod" {
		return stepError("chmod")
	}
	return nil
}

// Close ends the file, or fails at the close step.
func (m *memTemp) Close() error {
	if m.disk.failStep == "close" {
		return stepError("close")
	}
	return nil
}

// sinkOn returns the export sink over the in-memory folder, for the folder dir.
func sinkOn(d *memDisk, dir string) *markdownfiles.Sink {
	return markdownfiles.New(dir, d.create, markdownfiles.Folder{
		MkdirAll: d.mkdirAll,
		Rename:   d.rename,
		Remove:   d.remove,
	})
}

// hasTemp reports whether the folder holds a temporary export file.
func hasTemp(d *memDisk) bool {
	for name := range d.files {
		if strings.Contains(name, ".export-") {
			return true
		}
	}
	return false
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A write stores the file under the folder and returns its path
func TestAWriteStoresTheFileUnderTheFolderAndReturnsItsPath(t *testing.T) {
	d := newMemDisk()
	d.dirs["/exports"] = true
	path, err := sinkOn(d, "/exports").Write(context.Background(), "doc-1.md", []byte("# Notes"))
	if err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if path != "/exports/doc-1.md" {
		t.Fatalf("path = %q, want /exports/doc-1.md", path)
	}
	if got := d.files["/exports/doc-1.md"]; got != "# Notes" {
		t.Fatalf("file holds %q, want the bytes", got)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A write creates the folder when it is missing
func TestAWriteCreatesTheFolderWhenItIsMissing(t *testing.T) {
	d := newMemDisk()
	if _, err := sinkOn(d, "/exports/new").Write(context.Background(), "doc-1.md", []byte("# Notes")); err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if !d.dirs["/exports/new"] {
		t.Fatalf("folder was not created")
	}
	if got := d.files["/exports/new/doc-1.md"]; got != "# Notes" {
		t.Fatalf("file holds %q, want the bytes", got)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A write replaces an existing file and leaves no temporary file
func TestAWriteReplacesAnExistingFileAndLeavesNoTemporaryFile(t *testing.T) {
	d := newMemDisk()
	d.dirs["/exports"] = true
	d.files["/exports/doc-1.md"] = "old"
	if _, err := sinkOn(d, "/exports").Write(context.Background(), "doc-1.md", []byte("new")); err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if got := d.files["/exports/doc-1.md"]; got != "new" {
		t.Fatalf("file holds %q, want new", got)
	}
	if hasTemp(d) {
		t.Fatalf("a temporary file was left in the folder")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A file name that is not one plain .md name is refused before any folder
func TestAFileNameThatIsNotOnePlainMdNameIsRefusedBeforeAnyFolder(t *testing.T) {
	names := []string{"../doc.md", "sub/doc.md", "doc.txt", ".md", ".", "..", ""}
	for _, name := range names {
		t.Run("name "+name, func(t *testing.T) {
			d := newMemDisk()
			_, err := sinkOn(d, "/exports").Write(context.Background(), name, []byte("x"))
			wantError(t, err, "the export file name must be one plain name ending in .md")
			if len(d.files) != 0 || len(d.dirs) != 0 {
				t.Fatalf("the folder was touched: files %v, folders %v", d.files, d.dirs)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A sink with no folder refuses to write
func TestASinkWithNoFolderRefusesToWrite(t *testing.T) {
	d := newMemDisk()
	_, err := sinkOn(d, "").Write(context.Background(), "doc-1.md", []byte("x"))
	wantError(t, err, "no export folder was given")
	if len(d.files) != 0 {
		t.Fatalf("files = %v, want nothing written", d.files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A folder that cannot be created returns the folder error
func TestAFolderThatCannotBeCreatedReturnsTheFolderError(t *testing.T) {
	d := newMemDisk()
	d.failStep = "mkdir"
	_, err := sinkOn(d, "/exports").Write(context.Background(), "doc-1.md", []byte("x"))
	if err == nil || !strings.HasPrefix(err.Error(), "creating the export folder") {
		t.Fatalf("error = %v, want the folder error", err)
	}
	if len(d.files) != 0 {
		t.Fatalf("files = %v, want nothing written", d.files)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A failed step removes the temporary file and places no export
func TestAFailedStepRemovesTheTemporaryFileAndPlacesNoExport(t *testing.T) {
	steps := []struct {
		name string
		step string
	}{
		{"create the temporary file", "create"},
		{"write the bytes", "write"},
		{"set the mode", "chmod"},
		{"close the file", "close"},
		{"rename into place", "rename"},
	}
	for _, tc := range steps {
		t.Run(tc.name, func(t *testing.T) {
			d := newMemDisk()
			d.dirs["/exports"] = true
			d.failStep = tc.step
			_, err := sinkOn(d, "/exports").Write(context.Background(), "doc-1.md", []byte("x"))
			if err == nil {
				t.Fatalf("Write error = nil, want the failed step")
			}
			if _, ok := d.files["/exports/doc-1.md"]; ok {
				t.Fatalf("the export was placed although %s failed", tc.name)
			}
			if hasTemp(d) {
				t.Fatalf("a temporary file was left after %s failed", tc.name)
			}
		})
	}
}
