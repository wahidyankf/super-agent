package main

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/jiraseam"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// diskOf is the file tree of the path checks. The checks read it through the stat and lstat seams, so no test reads the
// working tree. The real disk is checked at the integration level, in disk_open_integration_test.go.
var diskOf = fstest.MapFS{
	"plain.txt": {Data: []byte("text")},
	"folder":    {Mode: fs.ModeDir | 0o755},
	"link":      {Mode: fs.ModeSymlink | 0o777, Data: []byte("plain.txt")},
}

// useDisk points the path checks at diskOf for the rest of the test.
func useDisk(t *testing.T) {
	t.Helper()
	oldStat, oldLstat := statPath, lstatPath
	statPath = func(name string) (fs.FileInfo, error) { return fs.Stat(diskOf, name) }
	lstatPath = lstatOf
	t.Cleanup(func() { statPath, lstatPath = oldStat, oldLstat })
}

// lstatOf describes the entry of diskOf itself, as os.Lstat describes a link: it never follows one. fstest.MapFS follows a
// link in Stat, so the entry is read from the map.
func lstatOf(name string) (fs.FileInfo, error) {
	f, ok := diskOf[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return entryInfo{name: name, f: f}, nil
}

// entryInfo is the fs.FileInfo of one entry of diskOf.
type entryInfo struct {
	name string
	f    *fstest.MapFile
}

func (e entryInfo) Name() string       { return e.name }
func (e entryInfo) Size() int64        { return int64(len(e.f.Data)) }
func (e entryInfo) Mode() fs.FileMode  { return e.f.Mode }
func (e entryInfo) ModTime() time.Time { return e.f.ModTime }
func (e entryInfo) IsDir() bool        { return e.f.Mode.IsDir() }
func (e entryInfo) Sys() any           { return nil }

// TestReadSeamRefusesAJiraSeamWithoutItsCAFile checks the error path of the Jira seam: a base without the CA file is
// refused, the same rule as the Outline seam.
func TestReadSeamRefusesAJiraSeamWithoutItsCAFile(t *testing.T) {
	getenv := mapEnv(map[string]string{jiraseam.BaseURLEnv: "https://127.0.0.1:1"})
	if _, err := readSeam(domain.KindJira, getenv); err == nil {
		t.Fatal("readSeam accepted a Jira seam without its CA file")
	}
}

// TestRegularFileExistsReportsOnlyRegularFiles checks the path test: a missing path is not a regular file, and neither is
// a folder or a link.
func TestRegularFileExistsReportsOnlyRegularFiles(t *testing.T) {
	useDisk(t)
	if !regularFileExists("plain.txt") {
		t.Fatal("regularFileExists() of a regular file is false")
	}
	if regularFileExists("definitely-not-a-retriever-file") {
		t.Fatal("regularFileExists() of a missing path is true")
	}
	if regularFileExists("folder") {
		t.Fatal("regularFileExists() of a folder is true")
	}
}

// TestRealDirExistsReportsOnlyRealFolders checks the folder check the profile resolution uses: a missing path, a file, and
// a link are not folders. The check describes the link itself, following nothing.
func TestRealDirExistsReportsOnlyRealFolders(t *testing.T) {
	useDisk(t)
	if !realDirExists("folder") {
		t.Fatal("realDirExists() of a folder is false")
	}
	if realDirExists("definitely-not-a-retriever-file") {
		t.Fatal("realDirExists() of a missing path is true")
	}
	if realDirExists("plain.txt") {
		t.Fatal("realDirExists() of a file is true")
	}
	if realDirExists("link") {
		t.Fatal("realDirExists() of a link is true")
	}
}

// TestRealSymlinkExistsReportsOnlyLinks checks the link check the profile resolution uses: a folder, a file, and a missing
// path are not links.
func TestRealSymlinkExistsReportsOnlyLinks(t *testing.T) {
	useDisk(t)
	if !realSymlinkExists("link") {
		t.Fatal("realSymlinkExists() of a link is false")
	}
	if realSymlinkExists("folder") {
		t.Fatal("realSymlinkExists() of a folder is true")
	}
	if realSymlinkExists("plain.txt") {
		t.Fatal("realSymlinkExists() of a file is true")
	}
	if realSymlinkExists("definitely-not-a-retriever-file") {
		t.Fatal("realSymlinkExists() of a missing path is true")
	}
}
