package memfs

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"testing"
	"time"
)

// mustWrite writes a file, creating its folder first, and fails the test on any error.
func mustWrite(t *testing.T, f *FS, name, text string) {
	t.Helper()
	if err := f.MkdirAll(path.Dir(name)); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path.Dir(name), err)
	}
	if err := f.WriteFile(name, []byte(text)); err != nil {
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
}

func TestWriteThenReadReturnsTheSameBytes(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/sources.yml", "version: 1\n")
	got, err := f.ReadFile("/scenario/sources.yml")
	if err != nil || string(got) != "version: 1\n" {
		t.Fatalf("ReadFile = %q, %v; want the written text", got, err)
	}
	if !f.IsRegular("/scenario/sources.yml") {
		t.Fatal("IsRegular reports no regular file for a written file")
	}
}

func TestMissingFileReadsAsNotExist(t *testing.T) {
	f := New()
	if _, err := f.ReadFile("/scenario/absent"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile of a missing file = %v; want fs.ErrNotExist", err)
	}
	if _, err := f.Lstat("/scenario/absent"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Lstat of a missing file = %v; want fs.ErrNotExist", err)
	}
	if f.IsRegular("/scenario/absent") {
		t.Fatal("IsRegular reports a missing file as present")
	}
}

func TestWriteFileNeedsItsFolder(t *testing.T) {
	f := New()
	if err := f.WriteFile("/scenario/none/a.md", []byte("x")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("WriteFile into a missing folder = %v; want fs.ErrNotExist", err)
	}
}

func TestPathsMustBeAbsoluteAndClean(t *testing.T) {
	f := New()
	if err := f.MkdirAll("scenario"); err == nil {
		t.Fatal("MkdirAll accepted a relative path")
	}
	if err := f.WriteFile("relative.md", nil); err == nil {
		t.Fatal("WriteFile accepted a relative path")
	}
	if _, err := f.ReadFile("scenario/x"); err == nil {
		t.Fatal("ReadFile accepted a relative path")
	}
	if _, err := f.Lstat("scenario/x"); err == nil {
		t.Fatal("Lstat accepted a relative path")
	}
	if _, err := f.Open("scenario/x"); err == nil {
		t.Fatal("Open accepted a relative path")
	}
	mustWrite(t, f, "/scenario/a/../b.md", "clean")
	if !f.IsRegular("/scenario/b.md") {
		t.Fatal("a path with dot segments was not cleaned")
	}
}

func TestAFileCannotBecomeAFolder(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/out", "text")
	if err := f.MkdirAll("/scenario/out/nested"); err == nil {
		t.Fatal("MkdirAll put a folder under a file")
	}
	if err := f.WriteFile("/scenario/out/x.md", nil); err == nil {
		t.Fatal("WriteFile put a file under a file")
	}
	if err := f.WriteFile("/scenario", nil); err == nil {
		t.Fatal("WriteFile replaced a folder")
	}
	if err := f.MkdirAll("/scenario/out"); err == nil {
		t.Fatal("MkdirAll replaced a file with a folder")
	}
}

func TestLstatReportsFoldersAsNotRegular(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/a/b.yml", "x")
	info, err := f.Lstat("/scenario/a")
	if err != nil {
		t.Fatalf("Lstat(folder): %v", err)
	}
	if !info.IsDir() || info.Mode().IsRegular() {
		t.Fatalf("a folder reports IsDir=%v regular=%v", info.IsDir(), info.Mode().IsRegular())
	}
	if f.IsRegular("/scenario/a") {
		t.Fatal("IsRegular reports a folder as a regular file")
	}
	if _, err := f.Open("/scenario/a"); err == nil {
		t.Fatal("Open opened a folder")
	}
}

func TestOpenedFileReadsAndStatsLikeTheRealOne(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/.env.local", "KEY=value\n")
	info, err := f.Lstat("/scenario/.env.local")
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Size() != int64(len("KEY=value\n")) || info.Name() != ".env.local" {
		t.Fatalf("Lstat reports name %q and size %d", info.Name(), info.Size())
	}
	file, err := f.Open("/scenario/.env.local")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	data, err := io.ReadAll(file)
	if err != nil || string(data) != "KEY=value\n" {
		t.Fatalf("read the opened file = %q, %v", data, err)
	}
	opened, err := file.Stat()
	if err != nil {
		t.Fatalf("Stat of the opened file: %v", err)
	}
	if !f.SameFile(info, opened) {
		t.Fatal("SameFile does not match a file with its own open description")
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSameFileTellsTwoFilesApart(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/a.yml", "a")
	mustWrite(t, f, "/scenario/b.yml", "a")
	a, _ := f.Lstat("/scenario/a.yml")
	b, _ := f.Lstat("/scenario/b.yml")
	if f.SameFile(a, b) {
		t.Fatal("SameFile matches two different files with equal contents")
	}
	if f.SameFile(a, fakeInfo{}) {
		t.Fatal("SameFile matches a description this tree did not make")
	}
	if f.SameFile(fakeInfo{}, a) {
		t.Fatal("SameFile matches a foreign description on the first argument")
	}
}

func TestWriteReplacesTheFileWithANewOne(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/out/a.md", "old")
	before, _ := f.Lstat("/scenario/out/a.md")
	if err := f.WriteFile("/scenario/out/a.md", []byte("new")); err != nil {
		t.Fatalf("WriteFile over an existing file: %v", err)
	}
	got, _ := f.ReadFile("/scenario/out/a.md")
	if string(got) != "new" {
		t.Fatalf("the file holds %q after a replacing write", got)
	}
	after, _ := f.Lstat("/scenario/out/a.md")
	if f.SameFile(before, after) {
		t.Fatal("a replaced file kept the identity of the old file")
	}
}

func TestWalkReturnsEveryFileUnderTheRootOnly(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/sources.yml", "one")
	mustWrite(t, f, "/scenario/out/deep/a.md", "two")
	mustWrite(t, f, "/other/b.md", "outside")
	files, err := f.Walk("/scenario")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	want := map[string]string{"sources.yml": "one", "out/deep/a.md": "two"}
	if len(files) != len(want) {
		t.Fatalf("Walk returned %d files, want %d: %v", len(files), len(want), files)
	}
	for name, text := range want {
		if string(files[name]) != text {
			t.Fatalf("Walk[%q] = %q, want %q", name, files[name], text)
		}
	}
}

func TestWalkOfAMissingRootIsEmpty(t *testing.T) {
	files, err := New().Walk("/scenario")
	if err != nil || len(files) != 0 {
		t.Fatalf("Walk of a missing root = %v, %v; want no files and no error", files, err)
	}
}

func TestFilesListsOnlyTheDirectChildren(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/out/a.md", "a")
	mustWrite(t, f, "/scenario/out/nested/b.md", "b")
	files, err := f.Files("/scenario/out")
	if err != nil {
		t.Fatalf("Files: %v", err)
	}
	if len(files) != 1 || string(files["a.md"]) != "a" {
		t.Fatalf("Files = %v; want only a.md", files)
	}
}

func TestFilesOfAMissingFolderIsEmpty(t *testing.T) {
	files, err := New().Files("/scenario/out")
	if err != nil || len(files) != 0 {
		t.Fatalf("Files of a missing folder = %v, %v; want none and no error", files, err)
	}
}

func TestFilesOfAFileIsAnError(t *testing.T) {
	f := New()
	mustWrite(t, f, "/scenario/a.md", "a")
	if _, err := f.Files("/scenario/a.md"); err == nil {
		t.Fatal("Files of a file returned no error")
	}
}

// fakeInfo is a file description this tree did not make.
type fakeInfo struct{}

func (fakeInfo) Name() string       { return "a.yml" }
func (fakeInfo) Size() int64        { return 1 }
func (fakeInfo) Mode() fs.FileMode  { return 0o644 }
func (fakeInfo) ModTime() time.Time { return time.Time{} }
func (fakeInfo) IsDir() bool        { return false }
func (fakeInfo) Sys() any           { return nil }
