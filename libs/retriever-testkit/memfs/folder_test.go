package memfs

import (
	"io"
	"io/fs"
	"testing"
)

// TestFolderNamesEntriesRelativeToTheFolder checks that a folder view reads and describes an entry by its name under the
// folder, and that the in-memory tree holds the entry the name reaches.
func TestFolderNamesEntriesRelativeToTheFolder(t *testing.T) {
	fsys := New()
	if err := fsys.MkdirAll("/work/project"); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := fsys.WriteFile("/work/project/sources.yml", []byte("version: 1")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	view := fsys.Folder("/work/project")
	info, err := fs.Stat(view, "sources.yml")
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("Stat(sources.yml) = %v, %v; want a regular file", info, err)
	}
	file, err := view.Open("sources.yml")
	if err != nil {
		t.Fatalf("Open(sources.yml): %v", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil || string(data) != "version: 1" {
		t.Fatalf("read = %q, %v; want the file content", data, err)
	}
	if _, err := fs.Stat(view, "missing.yml"); err == nil {
		t.Fatal("Stat of a missing entry returned no error")
	}
	if _, err := view.Open("missing.yml"); err == nil {
		t.Fatal("Open of a missing entry returned no error")
	}
}
