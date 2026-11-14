package main

import (
	"crypto/x509"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/catalog"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// host is what the composition root takes from the process and the disk: the standard streams, the working directory,
// the clock, the path test, the openers of the files and the connectors folder, the export sink, and the ports of the
// generic engine. main passes realHost; a unit test passes fakes, so the wiring runs without a real stream, file, folder,
// or socket.
type host struct {
	stdin          io.Reader
	stdout         io.Writer
	stderr         io.Writer
	getwd          func() (string, error)
	now            func() time.Time
	exists         func(path string) bool
	openEnvFile    func(path string) (*envfile.Credentials, error)
	openCatalog    func(path string, kinds domain.KindSet) (*catalog.Loader, error)
	openConnectors func(dir string) fs.FS
	exportSink     func(dir string) ports.ExportSink
	engine         engineSeam
}

// engineSeam holds the outbound ports of the generic engine for every spec source. realHost passes the real clock and
// the real network. A test passes fakes, a replay transport in place of the network, or a loopback network with the
// options that allow a private address. The transport and the options are empty in production.
type engineSeam struct {
	clock     engine.Clock
	network   engine.Network
	transport http.RoundTripper
	roots     *x509.CertPool
	options   []engine.Option
}

// realHost is the host of the real process. Only main passes it.
var realHost = host{
	stdin:          os.Stdin,
	stdout:         os.Stdout,
	stderr:         os.Stderr,
	getwd:          os.Getwd,
	now:            time.Now,
	exists:         regularFileExists,
	openEnvFile:    openEnvFile,
	openCatalog:    openCatalog,
	openConnectors: openConnectorsFolder,
	exportSink:     exportSink,
	engine:         engineSeam{clock: realClock{}, network: realNetwork()},
}
