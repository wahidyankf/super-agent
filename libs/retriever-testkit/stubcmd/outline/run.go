// Package outline is the Outline stub command as a library. Run parses the command line, starts the scripted Outline stub
// over TLS, writes the certificate the client must trust to the -ca-file, prints its loopback address, and waits until a
// stop signal arrives. The outlinestubcmd process is a call to Run with the real Host. A test passes fakes for the files,
// the clock, the TLS server, and the signals, so it runs the command without touching the machine.
//
// The command prints one line, "outline stub listening on <url>", once it listens. It refuses bad flags before it listens.
package outline

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// errNoCAFile reports a run that names no -ca-file.
var errNoCAFile = errors.New("the -ca-file flag is required")

// errTwoFaults reports a run that asks for two faults, which the stub holds one at a time.
var errTwoFaults = errors.New("only one of -list-status and -document-status can be set")

// Started is a TLS server the command started: the base URL it announces, the DER certificate that its clients trust, and
// the function that stops it.
type Started struct {
	URL  string
	DER  []byte
	Stop func()
}

// Host is everything the command reads from and writes to, apart from its two output streams. Real is the host of a
// process; a test passes fakes for every field it needs.
type Host struct {
	ReadFile  func(name string) ([]byte, error)
	WriteFile func(name string, data []byte, perm fs.FileMode) error
	Now       func() time.Time
	Serve     func(h http.Handler) (Started, error)
	Notify    func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)
}

// Real is the host of a process: the real files, clock, loopback socket, and stop signals.
var Real = Host{
	ReadFile:  os.ReadFile,
	WriteFile: os.WriteFile,
	Now:       time.Now,
	Serve:     serveTLSWith(net.Listen, time.Now),
	Notify:    signal.NotifyContext,
}

// serveTLSWith returns the Serve seam that starts h over TLS on the listener that listen opens, with a certificate dated by
// now. The real host passes the process socket and clock; a test passes an in-memory listener.
func serveTLSWith(listen func(network, address string) (net.Listener, error), now func() time.Time) func(http.Handler) (Started, error) {
	return func(h http.Handler) (Started, error) {
		srv, err := outlinestub.NewTLSServerWith(h, listen, now)
		if err != nil {
			return Started{}, err
		}
		return Started{URL: srv.URL, DER: srv.Certificate().Raw, Stop: srv.Close}, nil
	}
}

// config is what the command needs to start the stub.
type config struct {
	caFile         string
	collections    []outlinestub.Collection
	documents      []outlinestub.Document
	listStatus     int
	documentStatus int
}

// readDocuments reads the documents the stub holds from a JSON file: an array of objects with the Document field
// names, such as {"ID":"doc-2","CollectionID":"c1","ParentID":"","Title":"Setup","URL":"/doc/doc-2","Markdown":"..."}.
func readDocuments(readFile func(name string) ([]byte, error), path string) ([]outlinestub.Document, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("read the documents file: %w", err)
	}
	var docs []outlinestub.Document
	if err := json.Unmarshal(data, &docs); err != nil {
		return nil, fmt.Errorf("parse the documents file: %w", err)
	}
	return docs, nil
}

// start serves the stub over TLS through h.Serve, and writes the certificate to cfg.caFile. A fault on collections.list
// applies when cfg.listStatus is set. The caller stops the returned server.
func start(cfg config, h Host) (Started, error) {
	if cfg.listStatus > 0 && cfg.documentStatus > 0 {
		return Started{}, errTwoFaults
	}
	stub := outlinestub.New(
		outlinestub.WithCollections(cfg.collections...),
		outlinestub.WithDocuments(cfg.documents...),
		outlinestub.WithClock(h.Now),
	)
	if cfg.listStatus > 0 {
		stub.SetFault(outlinestub.Fault{Method: "collections.list", Status: cfg.listStatus})
	}
	if cfg.documentStatus > 0 {
		stub.SetFault(outlinestub.Fault{Method: "documents.info", Status: cfg.documentStatus})
	}
	server, err := h.Serve(stub)
	if err != nil {
		return Started{}, fmt.Errorf("start the TLS stub: %w", err)
	}
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.DER})
	if err := h.WriteFile(cfg.caFile, certificate, 0o600); err != nil {
		server.Stop()
		return Started{}, fmt.Errorf("write the CA file: %w", err)
	}
	return server, nil
}

// serve parses the flags, starts the stub, prints its address to stdout, and waits until ctx is done. Flag diagnostics go
// to stderr.
func serve(ctx context.Context, args []string, stdout, stderr io.Writer, h Host) error {
	flags := flag.NewFlagSet("outlinestubcmd", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var collections collectionFlag
	caFile := flags.String("ca-file", "", "where to write the PEM certificate of the stub")
	listStatus := flags.Int("list-status", 0, "answer collections.list with this HTTP status; 0 replays normally")
	documentStatus := flags.Int("document-status", 0, "answer documents.info with this HTTP status; 0 replays normally")
	documentsFile := flags.String("documents-file", "", "a JSON file of the documents the stub holds")
	flags.Var(&collections, "collection", "one collection as id=name; repeat it in listing order")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *caFile == "" {
		return errNoCAFile
	}

	var documents []outlinestub.Document
	if *documentsFile != "" {
		docs, err := readDocuments(h.ReadFile, *documentsFile)
		if err != nil {
			return err
		}
		documents = docs
	}

	server, err := start(config{
		caFile:         *caFile,
		collections:    collections,
		documents:      documents,
		listStatus:     *listStatus,
		documentStatus: *documentStatus,
	}, h)
	if err != nil {
		return err
	}
	defer server.Stop()

	fmt.Fprintf(stdout, "outline stub listening on %s\n", server.URL)

	<-ctx.Done()
	return nil
}

// Run runs the command with args and returns its exit status: 0 when the stub stopped on a stop signal, and 1 when it
// refused to run, after it printed the refusal to stderr.
func Run(args []string, stdout, stderr io.Writer, h Host) int {
	ctx, stop := h.Notify(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, args, stdout, stderr, h); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
