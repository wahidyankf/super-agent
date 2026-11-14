package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
	"github.com/wahidyankf/super-agent/libs/retriever-testkit/sourcecontract"
)

// memSink is an in-memory ports.ExportSink. It keeps the file names and bodies a test exports, and can fail every write.
type memSink struct {
	dir    string
	names  []string
	bodies map[string][]byte
	err    error
}

func newMemSink() *memSink {
	return &memSink{bodies: map[string][]byte{}}
}

func (s *memSink) Write(_ context.Context, name string, data []byte) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	s.names = append(s.names, name)
	s.bodies[name] = data
	return s.dir + "/" + name, nil
}

// countingGateway counts the calls made through it, so a test can prove a refused request never reached the source.
type countingGateway struct {
	inner sourcecontract.Gateway
	calls int
}

func (g *countingGateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	g.calls++
	return g.inner.ListCollections(ctx)
}

func (g *countingGateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	g.calls++
	return g.inner.CollectionTree(ctx, id)
}

func (g *countingGateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	g.calls++
	return g.inner.Document(ctx, id)
}

func (g *countingGateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	g.calls++
	return g.inner.Read(ctx, op, args)
}

// failingGateway answers every call with one fixed error.
type failingGateway struct {
	err error
}

func (g failingGateway) ListCollections(context.Context) ([]domain.Collection, error) {
	return nil, g.err
}

func (g failingGateway) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	return nil, g.err
}

func (g failingGateway) Document(context.Context, domain.DocumentID) (domain.SourceDocument, error) {
	return domain.SourceDocument{}, g.err
}

func (g failingGateway) Read(context.Context, domain.OperationID, domain.Args) (domain.Record, error) {
	return domain.Record{}, g.err
}

// outlineSetup is the wiring a test gives the CLI in place of the composition root's.
type outlineSetup struct {
	service  sources.Service
	sink     *memSink
	setupErr error
	serve    ServeFunc
	// ctx is the parent context of serve and mcp. A test ends a serve command as a signal would, by cancelling it with
	// SignalCause. Nil means Background.
	ctx context.Context
}

// wikiWith returns a setup whose Outline gateway serves dataset and whose export sink is empty.
func wikiWith(dataset sourcecontract.Dataset) (outlineSetup, *countingGateway) {
	gw := &countingGateway{inner: sourcecontract.NewFake(sourcecontract.Wiring{Dataset: dataset})}
	return outlineSetup{
		service: oneOutlineService(gw),
		sink:    newMemSink(),
	}, gw
}

// runRetriever runs argv through Run against setup, and returns the exit code and the two output streams.
func runRetriever(t *testing.T, argv []string, setup outlineSetup) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	deps := Deps{
		Ports:    ports.Ports{FS: stubFS{}},
		Stdout:   &stdout,
		Stderr:   &stderr,
		Getenv:   func(string) string { return "" },
		Now:      func() time.Time { return fixedNow },
		Service:  setup.service,
		Serve:    setup.serve,
		SetupErr: setup.setupErr,
		Context:  setup.ctx,
	}
	if setup.sink != nil {
		deps.Sink = func(dir string) ports.ExportSink {
			setup.sink.dir = dir
			return setup.sink
		}
	}
	code := Run(argv, deps)
	return code, stdout.String(), stderr.String()
}

// indentOf returns the leading spaces of the first output line that contains text, or -1 when no line does.
func indentOf(out, text string) int {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, text) {
			return len(line) - len(strings.TrimLeft(line, " "))
		}
	}
	return -1
}

// treeDataset is one collection with a three-level tree: Getting Started, Setup under it, Install under Setup.
var treeDataset = sourcecontract.Dataset{
	Collections: []sourcecontract.Collection{{ID: "col-1", Name: "Engineering"}, {ID: "col-empty", Name: "Empty"}},
	Documents: []sourcecontract.Document{
		{ID: "doc-1", CollectionID: "col-1", Title: "Getting Started", URL: "/doc/getting-started", Markdown: "# Getting Started\n\nWelcome to the wiki.\n"},
		{ID: "doc-2", CollectionID: "col-1", ParentID: "doc-1", Title: "Setup", URL: "/doc/setup", Markdown: "# Setup\n"},
		{ID: "doc-3", CollectionID: "col-1", ParentID: "doc-2", Title: "Install", URL: "/doc/install", Markdown: "# Install\n"},
	},
}
