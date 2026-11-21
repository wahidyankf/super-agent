package unit_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/memfs"
)

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: The root folder is always present
func TestRootFolderIsAlwaysPresent(t *testing.T) {
	tree := memfs.New()
	info, err := tree.Lstat("/")
	if err != nil {
		t.Fatalf("describe the root folder: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("the root folder is not a folder")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A written file reads back with the same bytes
func TestWrittenFileReadsBackWithSameBytes(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work"); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/sources.yml", []byte("version: 1")); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := tree.ReadFile("/work/sources.yml")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "version: 1" {
		t.Fatalf("bytes read back = %q, want %q", got, "version: 1")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A write that replaces a file keeps the new bytes only
func TestWriteThatReplacesFileKeepsNewBytesOnly(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work"); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/sources.yml", []byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/sources.yml", []byte("new")); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, err := tree.ReadFile("/work/sources.yml")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("the file holds %q, want %q", got, "new")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A relative path is refused
func TestRelativePathIsRefused(t *testing.T) {
	for _, path := range []string{"work/x.yml", "work/../x"} {
		t.Run(path, func(t *testing.T) {
			got, err := memfs.New().ReadFile(path)
			if err == nil {
				t.Fatalf("read %q: no error", path)
			}
			if got != nil {
				t.Fatalf("read %q returned bytes %q with its error", path, got)
			}
		})
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A missing file reads as not existing
func TestMissingFileReadsAsNotExisting(t *testing.T) {
	_, err := memfs.New().ReadFile("/work/missing.yml")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error %v does not match not exist", err)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A file cannot be written where a folder is
func TestFileCannotBeWrittenWhereFolderIs(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work"); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work", []byte("x")); err == nil {
		t.Fatal("writing a file over a folder returned no error")
	}
	info, err := tree.Lstat("/work")
	if err != nil || !info.IsDir() {
		t.Fatalf("the folder changed: info %v, error %v", info, err)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A file cannot take a child
func TestFileCannotTakeChild(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work"); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/a.yml", []byte("keep")); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/a.yml/child", []byte("x")); err == nil {
		t.Fatal("writing a child under a file returned no error")
	}
	got, err := tree.ReadFile("/work/a.yml")
	if err != nil || string(got) != "keep" {
		t.Fatalf("the file changed: %q, error %v", got, err)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: The stat of a folder is not a regular file, and the stat of a file is
func TestStatOfFolderAndFileRegularity(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work"); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/a.yml", []byte("x")); err != nil {
		t.Fatal(err)
	}
	folder, err := tree.Lstat("/work")
	if err != nil {
		t.Fatal(err)
	}
	file, err := tree.Lstat("/work/a.yml")
	if err != nil {
		t.Fatal(err)
	}
	if folder.Mode().IsRegular() {
		t.Fatal("/work is a regular file")
	}
	if !file.Mode().IsRegular() {
		t.Fatal("/work/a.yml is not a regular file")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A walk of the root returns every file under it and no folder
func TestWalkOfRootReturnsEveryFileAndNoFolder(t *testing.T) {
	tree := memfs.New()
	for _, dir := range []string{"/a/b", "/empty"} {
		if err := tree.MkdirAll(dir); err != nil {
			t.Fatal(err)
		}
	}
	if err := tree.WriteFile("/a/one.yml", []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/a/b/two.yml", []byte("2")); err != nil {
		t.Fatal(err)
	}
	got, err := tree.Walk("/")
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(got) != 2 || string(got["a/one.yml"]) != "1" || string(got["a/b/two.yml"]) != "2" {
		t.Fatalf("walk returned %v, want a/one.yml and a/b/two.yml only", got)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A walk of a missing root returns nothing
func TestWalkOfMissingRootReturnsNothing(t *testing.T) {
	got, err := memfs.New().Walk("/missing")
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("walk returned %v, want no entries", got)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A folder view names its entries relative to the folder
func TestFolderViewNamesEntriesRelativeToFolder(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work/sub"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"/work/a.yml", "/work/b.yml"} {
		if err := tree.WriteFile(name, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	view := tree.Folder("/work")
	for _, name := range []string{"a.yml", "b.yml"} {
		info, err := fs.Stat(view, name)
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("%s is not a regular file: %v", name, err)
		}
	}
	info, err := fs.Stat(view, "sub")
	if err != nil || !info.IsDir() {
		t.Fatalf("sub is not a folder: %v", err)
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: A folder view of a missing folder reads nothing
func TestFolderViewOfMissingFolderReadsNothing(t *testing.T) {
	view := memfs.New().Folder("/missing")
	if _, err := fs.Stat(view, "a.yml"); err == nil {
		t.Fatal("a file of a missing folder was read")
	}
	if _, err := view.Open("a.yml"); err == nil {
		t.Fatal("a file of a missing folder was opened")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: Listing a folder names its direct children only
func TestListingFolderNamesDirectChildrenOnly(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work/sub"); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/a.yml", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/sub/b.yml", []byte("y")); err != nil {
		t.Fatal(err)
	}
	got, err := tree.Files("/work")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if _, ok := got["a.yml"]; !ok {
		t.Fatal("the list does not hold a.yml")
	}
	if _, ok := got["b.yml"]; ok {
		t.Fatal("the list holds b.yml, which is in a sub folder")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: Listing a missing folder is empty and listing a file is an error
func TestListingMissingFolderIsEmptyAndListingFileIsError(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work"); err != nil {
		t.Fatal(err)
	}
	if err := tree.WriteFile("/work/a.yml", []byte("x")); err != nil {
		t.Fatal(err)
	}
	missing, err := tree.Files("/missing")
	if err != nil || len(missing) != 0 {
		t.Fatalf("list of /missing = %v, error %v; want empty and no error", missing, err)
	}
	if _, err := tree.Files("/work/a.yml"); err == nil {
		t.Fatal("listing a file returned no error")
	}
}

// @scenario specs/libs/retriever-testkit/behaviours/in-memory-file-tree.feature: Two files that are written apart are not the same file
func TestTwoFilesWrittenApartAreNotTheSameFile(t *testing.T) {
	tree := memfs.New()
	if err := tree.MkdirAll("/work"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"/work/a.yml", "/work/b.yml"} {
		if err := tree.WriteFile(name, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	a, err := tree.Lstat("/work/a.yml")
	if err != nil {
		t.Fatal(err)
	}
	b, err := tree.Lstat("/work/b.yml")
	if err != nil {
		t.Fatal(err)
	}
	if tree.SameFile(a, b) {
		t.Fatal("two files that were written apart are the same file")
	}
}
