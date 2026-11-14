package memfs

import (
	"io/fs"
	"testing"
)

// TestRootFolderIsAlwaysThere checks that the root folder exists from the start, that making it again changes nothing,
// and that it cannot be replaced by a file or read as one.
func TestRootFolderIsAlwaysThere(t *testing.T) {
	f := New()
	if err := f.MkdirAll("/"); err != nil {
		t.Fatalf("MkdirAll(/): %v", err)
	}
	info, err := f.Lstat("/")
	if err != nil || !info.IsDir() {
		t.Fatalf("Lstat(/) = %v, %v; want the root folder", info, err)
	}
	if err := f.WriteFile("/", []byte("x")); err == nil {
		t.Fatal("WriteFile replaced the root folder with a file")
	}
	if _, err := f.ReadFile("/"); err == nil {
		t.Fatal("ReadFile read the root folder as a file")
	}
}

// TestAFolderIsNotAFile checks that a folder is refused by each call that needs a regular file.
func TestAFolderIsNotAFile(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/a.md", "a")
	if _, err := f.ReadFile("/scenario"); err == nil {
		t.Fatal("ReadFile read a folder as a file")
	}
	if f.IsRegular("/scenario") {
		t.Fatal("IsRegular reports a folder as a regular file")
	}
}

// TestRelativePathsAreRefused checks that the calls which take a folder or a file refuse a relative path, so that a
// name can only mean one entry of the tree.
func TestRelativePathsAreRefused(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/a.md", "a")
	if f.IsRegular("scenario/a.md") {
		t.Fatal("IsRegular accepted a relative path")
	}
	if _, err := f.Files("scenario"); err == nil {
		t.Fatal("Files accepted a relative path")
	}
	if _, err := f.Walk("scenario"); err == nil {
		t.Fatal("Walk accepted a relative path")
	}
}

// TestWalkOfTheRootReturnsEveryFile checks that walking "/" keys each file by its path under the root, with no doubled
// slash in front.
func TestWalkOfTheRootReturnsEveryFile(t *testing.T) {
	f := New()
	mustWrite(t, f, "/a.md", "one")
	mustWrite(t, f, "/x/b.md", "two")
	files, err := f.Walk("/")
	if err != nil {
		t.Fatalf("Walk(/): %v", err)
	}
	want := map[string]string{"a.md": "one", "x/b.md": "two"}
	if len(files) != len(want) {
		t.Fatalf("Walk(/) returned %d files, want %d: %v", len(files), len(want), files)
	}
	for name, text := range want {
		if string(files[name]) != text {
			t.Fatalf("Walk(/)[%q] = %q, want %q", name, files[name], text)
		}
	}
}

// TestEntryDescriptionsHaveNoClockAndTheirOwnIdentity checks that an entry's modification time is the zero time, as the
// tree keeps no clock, and that the identity each description carries belongs to its own entry.
func TestEntryDescriptionsHaveNoClockAndTheirOwnIdentity(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/a.yml", "a")
	mustWrite(t, f, "/scenario/b.yml", "b")
	a, err := f.Lstat("/scenario/a.yml")
	if err != nil {
		t.Fatalf("Lstat(a.yml): %v", err)
	}
	b, err := f.Lstat("/scenario/b.yml")
	if err != nil {
		t.Fatalf("Lstat(b.yml): %v", err)
	}
	if !a.ModTime().IsZero() {
		t.Fatalf("ModTime = %v, want the zero time", a.ModTime())
	}
	if a.Sys() == nil {
		t.Fatal("Sys holds no identity")
	}
	if a.Sys() == b.Sys() {
		t.Fatal("two different files share one identity")
	}
	if !f.SameFile(a, a) {
		t.Fatal("SameFile does not match an entry with itself")
	}
}

// TestFolderViewOfAMissingFolderReadsNothing checks that a folder view over a folder that does not exist reports each of
// its names as missing, the way a real missing folder would.
func TestFolderViewOfAMissingFolderReadsNothing(t *testing.T) {
	view := New().Folder("/absent")
	if _, err := fs.Stat(view, "a.yml"); err == nil {
		t.Fatal("Stat of a name in a missing folder returned no error")
	}
}
