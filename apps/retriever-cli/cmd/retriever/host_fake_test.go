package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net"
	"net/netip"
	"strings"
	"testing/fstest"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
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
	return hostOver(getwd, fstest.MapFS{}, fstest.MapFS{})
}

// hostOver returns the unit-test host whose catalog and env file are read from files, and whose connectors folder is
// connectors. The working directory is getwd. Both folders are in memory.
func hostOver(getwd func() (string, error), files fstest.MapFS, connectors fs.FS) host {
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
		openConnectors: func(string) fs.FS { return connectors },
		exportSink:     func(string) ports.ExportSink { return nil },
		engine:         testEngineSeam(),
	}
	return h
}

// testEngineSeam returns the engine ports of a unit test: a clock that never waits, and a network that refuses every call,
// so no spec source reaches a socket. A test that runs a spec source sets the transport to a replay stub.
func testEngineSeam() engineSeam {
	return engineSeam{
		clock: fakeClock{},
		network: engine.Network{
			Lookup: func(context.Context, string) ([]netip.Addr, error) { return nil, errNoNetwork },
			Dial:   func(context.Context, string, string) (net.Conn, error) { return nil, errNoNetwork },
		},
	}
}

// errNoNetwork is the failure of the test network, which every call returns.
var errNoNetwork = errors.New("the unit-test network refuses every call")

// fakeClock is the clock of a unit test. A time limit ends when its parent ends, and a wait returns at once, so no test
// waits on a timer.
type fakeClock struct{}

// WithTimeout returns a context that ends only when its parent ends, or when the release function runs.
func (fakeClock) WithTimeout(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// Sleep returns at once, unless ctx has already ended.
func (fakeClock) Sleep(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}

// sameName reports whether two in-memory descriptions name the same entry. The in-memory tree has no links, so the name
// decides.
func sameName(a, b fs.FileInfo) bool { return a.Name() == b.Name() }
