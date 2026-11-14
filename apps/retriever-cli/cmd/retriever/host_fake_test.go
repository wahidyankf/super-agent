package main

import (
	"bytes"
	"io/fs"
	"strings"
	"testing/fstest"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// testHost returns the host of a unit test. Every stream is a buffer, the working directory is a fixed folder, no path
// exists, and both files are read from an empty in-memory folder. The export sink returns nothing, which no unit test
// calls. No real stream, file, or folder is touched.
func testHost() host {
	return testHostWithGetwd(func() (string, error) { return "/work/project", nil })
}

// testHostWithGetwd returns the unit-test host with the given working directory.
func testHostWithGetwd(getwd func() (string, error)) host {
	files := fstest.MapFS{}
	h := host{
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		getwd:  getwd,
		now:    func() time.Time { return time.Unix(0, 0) },
		exists: func(string) bool { return false },
		openEnvFile: func(path string) (*envfile.Credentials, error) {
			return envfile.NewWithFiles(path, func(string) string { return "" }, envfile.Files{FS: files, SameFile: sameName})
		},
		openCatalog: func(path string, kinds domain.KindSet) (*catalog.Loader, error) {
			return catalog.NewWithFiles(path, catalog.Files{FS: files, SameFile: sameName}, kinds)
		},
		exportSink: func(string) ports.ExportSink { return nil },
	}
	return h
}

// sameName reports whether two in-memory descriptions name the same entry. The in-memory tree has no links, so the name
// decides.
func sameName(a, b fs.FileInfo) bool { return a.Name() == b.Name() }
