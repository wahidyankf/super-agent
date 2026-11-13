// Package main is a test-only command. It serves the scripted Outline stub over TLS on a loopback port, as a process,
// so the Elixir end-to-end layer can point the real retriever binary at it through the loopback seam of ADR 003. It is
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
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
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

// readDocuments reads the documents the stub holds from a JSON file: an array of objects with the Document field
// names, such as {"ID":"doc-2","CollectionID":"c1","ParentID":"","Title":"Setup","URL":"/doc/doc-2","Markdown":"..."}.
func readDocuments(path string) ([]outlinestub.Document, error) {
	data, err := os.ReadFile(path)
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

// start serves the stub over TLS on a loopback port, and writes the certificate to cfg.caFile. A fault on
// collections.list applies when cfg.listStatus is set. The caller closes the returned server.
func start(cfg config) (*httptest.Server, error) {
	if cfg.listStatus > 0 && cfg.documentStatus > 0 {
		return nil, errTwoFaults
	}
	stub := outlinestub.New(
		outlinestub.WithCollections(cfg.collections...),
		outlinestub.WithDocuments(cfg.documents...),
	)
	if cfg.listStatus > 0 {
		stub.SetFault(outlinestub.Fault{Method: "collections.list", Status: cfg.listStatus})
	}
	if cfg.documentStatus > 0 {
		stub.SetFault(outlinestub.Fault{Method: "documents.info", Status: cfg.documentStatus})
	}
	server, err := outlinestub.NewTLSServer(stub)
	if err != nil {
		return nil, fmt.Errorf("make the TLS certificate: %w", err)
	}
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(cfg.caFile, certificate, 0o600); err != nil {
		server.Close()
		return nil, fmt.Errorf("write the CA file: %w", err)
	}
	return server, nil
}

// run parses the flags, starts the stub, prints its address, and waits until ctx is done or a stop signal arrives.
func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("outlinestubcmd", flag.ContinueOnError)
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
		docs, err := readDocuments(*documentsFile)
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
	})
	if err != nil {
		return err
	}
	defer server.Close()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(out, "outline stub listening on %s\n", server.URL)

	<-ctx.Done()
	return nil
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
