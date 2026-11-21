//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hasTempFile reports whether dir holds a temporary export file, which the sink names with the prefix .export-.
func hasTempFile(t *testing.T, dir string) bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the export folder: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".export-") {
			return true
		}
	}
	return false
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A write stores the file under the folder and returns its path
func TestAWriteStoresTheFileUnderTheFolderAndReturnsItsPath(t *testing.T) {
	dir := t.TempDir()
	path, err := realSink(dir).Write(context.Background(), "doc-1.md", []byte("# Notes"))
	if err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if want := filepath.Join(dir, "doc-1.md"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "# Notes" {
		t.Fatalf("file holds %q, error %v; want the bytes", data, err)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A write creates the folder when it is missing
func TestAWriteCreatesTheFolderWhenItIsMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	if _, err := realSink(dir).Write(context.Background(), "doc-1.md", []byte("# Notes")); err != nil {
		t.Fatalf("Write error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "doc-1.md"))
	if err != nil || string(data) != "# Notes" {
		t.Fatalf("file holds %q, error %v; want the bytes in the new folder", data, err)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A write replaces an existing file and leaves no temporary file
func TestAWriteReplacesAnExistingFileAndLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"doc-1.md": "old"})
	if _, err := realSink(dir).Write(context.Background(), "doc-1.md", []byte("new")); err != nil {
		t.Fatalf("Write error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "doc-1.md"))
	if err != nil || string(data) != "new" {
		t.Fatalf("file holds %q, error %v; want new", data, err)
	}
	if hasTempFile(t, dir) {
		t.Fatalf("a temporary file was left in the folder")
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A file name that is not one plain .md name is refused before any folder
func TestAFileNameThatIsNotOnePlainMdNameIsRefusedBeforeAnyFolder(t *testing.T) {
	names := []string{"../doc.md", "sub/doc.md", "doc.txt", ".md", ".", "..", ""}
	for _, name := range names {
		t.Run("name "+name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "exports")
			_, err := realSink(dir).Write(context.Background(), name, []byte("x"))
			wantError(t, err, "the export file name must be one plain name ending in .md")
			if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
				t.Fatalf("the export folder was created although the name was refused")
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/export-sink.feature: A folder that cannot be created returns the folder error
func TestAFolderThatCannotBeCreatedReturnsTheFolderError(t *testing.T) {
	// A regular file where the export folder should be makes the folder impossible to create on the real disk.
	blocker := filepath.Join(t.TempDir(), "blocker")
	writeFiles(t, filepath.Dir(blocker), map[string]string{"blocker": "not a folder"})
	_, err := realSink(filepath.Join(blocker, "exports")).Write(context.Background(), "doc-1.md", []byte("x"))
	if err == nil || !strings.HasPrefix(err.Error(), "creating the export folder") {
		t.Fatalf("error = %v, want the folder error", err)
	}
}
