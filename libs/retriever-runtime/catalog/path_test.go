package catalog

import "testing"

// TestNewWithFilesRefusesARelativePath checks that a loader is never built for a path that is not absolute, because the
// catalog folder is the root of the file calls.
func TestNewWithFilesRefusesARelativePath(t *testing.T) {
	if _, err := NewWithFiles("sources.yml", Files{FS: fakeFS{}, SameFile: sameFake}); err == nil {
		t.Fatal("NewWithFiles of a relative path returned no error")
	}
}
