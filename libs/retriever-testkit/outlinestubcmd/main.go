// Package main is a test-only command. It serves the scripted Outline stub over TLS on a loopback port, as a process,
// so the Elixir end-to-end layer can point the real retriever binary at it through the loopback seam. It is
// not part of the retriever binary, and nothing in retriever-cli depends on it.
//
// The command prints one line, "outline stub listening on <url>", once it listens. It writes the PEM certificate
// the client must trust to -ca-file before that line. It runs until it receives SIGINT or SIGTERM.
package main

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
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-testkit/outlinestub"
)

// errNoCAFile reports a run that names no -ca-file.
var errNoCAFile = errors.New("the -ca-file flag is required")

// errTwoFaults reports a run that asks for two faults, which the stub holds one at a time.
var errTwoFaults = errors.New("only one of -list-status and -document-status can be set")

// config is what the command needs to start the stub.
type config struct {
	caFile         string
	collections    []outlinestub.Collection
	documents      []outlinestub.Document
	listStatus     int
	documentStatus int
}

// stdio is where the command writes its listening line, and where it writes its messages when it refuses to run.
type stdio struct {
	out io.Writer
	err io.Writer
}

// deps are the effects the command has: the files it reads and writes, the listener it serves on, the clock that dates
// the certificate, and the stop signals it waits for. main supplies the real ones, and a test supplies fakes.
type deps struct {
	readFile  func(name string) ([]byte, error)
	writeFile func(name string, data []byte, perm fs.FileMode) error
	listen    func(network, address string) (net.Listener, error)
	now       func() time.Time
	notify    func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc)
}

// system holds the real effects that main runs with.
var system = deps{
	readFile:  os.ReadFile,
	writeFile: os.WriteFile,
	listen:    net.Listen,
	now:       time.Now,
	notify:    signal.NotifyContext,
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

// collectionFlag collects repeated -collection id=name flags in listing order.
type collectionFlag []outlinestub.Collection

// String implements flag.Value.
func (c *collectionFlag) String() string { return fmt.Sprint(*c) }

// Set implements flag.Value. The value splits at its first "=", so a name may hold "=". Both parts must be non-empty.
func (c *collectionFlag) Set(value string) error {
	id, name, found := strings.Cut(value, "=")
	if !found || id == "" || name == "" {
		return fmt.Errorf("-collection wants id=name, got %q", value)
	}
	*c = append(*c, outlinestub.Collection{ID: id, Name: name})
	return nil
}

// start serves the stub over TLS on the listener d opens, and writes the certificate to cfg.caFile. A fault on
// collections.list applies when cfg.listStatus is set. The caller closes the returned server.
func start(cfg config, d deps) (*httptest.Server, error) {
	if cfg.listStatus > 0 && cfg.documentStatus > 0 {
		return nil, errTwoFaults
	}
	stub := outlinestub.New(
		outlinestub.WithCollections(cfg.collections...),
		outlinestub.WithDocuments(cfg.documents...),
		outlinestub.WithClock(d.now),
	)
	if cfg.listStatus > 0 {
		stub.SetFault(outlinestub.Fault{Method: "collections.list", Status: cfg.listStatus})
	}
	if cfg.documentStatus > 0 {
		stub.SetFault(outlinestub.Fault{Method: "documents.info", Status: cfg.documentStatus})
	}
	server, err := outlinestub.NewTLSServerWith(stub, d.listen, d.now)
	if err != nil {
		return nil, fmt.Errorf("start the TLS stub: %w", err)
	}
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := d.writeFile(cfg.caFile, certificate, 0o600); err != nil {
		server.Close()
		return nil, fmt.Errorf("write the CA file: %w", err)
	}
	return server, nil
}

// run starts the command with args and returns its exit code: 0 when the stub stopped on a stop signal, and 1 when it
// refused to run, after it printed the refusal to std.err.
func run(args []string, std stdio, d deps) int {
	ctx, stop := d.notify(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, args, std, d); err != nil {
		fmt.Fprintln(std.err, err)
		return 1
	}
	return 0
}

// serve parses the flags, starts the stub, prints its address, and waits until ctx is done.
func serve(ctx context.Context, args []string, std stdio, d deps) error {
	flags := flag.NewFlagSet("outlinestubcmd", flag.ContinueOnError)
	flags.SetOutput(std.err)
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
		docs, err := readDocuments(d.readFile, *documentsFile)
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
	}, d)
	if err != nil {
		return err
	}
	defer server.Close()

	fmt.Fprintf(std.out, "outline stub listening on %s\n", server.URL)

	<-ctx.Done()
	return nil
}

// process is what main takes from the real process: its arguments, its standard streams, its exit, and the effects of
// the run. main reads it in one statement, and a test replaces it, so main itself runs with fakes.
type process struct {
	args    []string
	std     stdio
	exit    func(code int)
	effects deps
}

// proc is the real process: its own arguments and streams, the real exit, and the real effects.
var proc = process{
	args:    os.Args[1:],
	std:     stdio{out: os.Stdout, err: os.Stderr},
	exit:    os.Exit,
	effects: system,
}

func main() {
	proc.exit(run(proc.args, proc.std, proc.effects))
}
