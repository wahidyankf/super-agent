package main

import (
	"io"
	"os"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// host is what the composition root takes from the process and the disk: the standard streams, the working directory,
// the clock, the path test, the openers of the two files, and the export sink. main passes realHost; a unit test passes
// fakes, so the wiring runs without a real stream, file, or folder.
type host struct {
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	getwd       func() (string, error)
	now         func() time.Time
	exists      func(path string) bool
	openEnvFile func(path string) (*envfile.Credentials, error)
	openCatalog func(path string, kinds domain.KindSet) (*catalog.Loader, error)
	exportSink  func(dir string) ports.ExportSink
}

// realHost is the host of the real process. Only main passes it.
var realHost = host{
	stdin:       os.Stdin,
	stdout:      os.Stdout,
	stderr:      os.Stderr,
	getwd:       os.Getwd,
	now:         time.Now,
	exists:      regularFileExists,
	openEnvFile: openEnvFile,
	openCatalog: openCatalog,
	exportSink:  exportSink,
}
