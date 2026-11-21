package unit_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// This file holds the doubles that the unit bindings share. It has no scenario of its own. Each double answers from memory,
// so no test reads a disk, a variable, a clock, or a process.

// sameByName reports whether two descriptions name the same file. An in-memory folder holds one entry per name, so the name
// is enough to tell that the file was not swapped while it was opened.
func sameByName(a, b fs.FileInfo) bool { return a.Name() == b.Name() }

// lookup returns a getenv over the given variables. A variable that is not in the map reads as "", as an unset process
// variable does.
func lookup(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// countingLookup returns a getenv over vars that also counts each read into reads.
func countingLookup(vars map[string]string, reads *[]string) func(string) string {
	return func(name string) string {
		*reads = append(*reads, name)
		return vars[name]
	}
}

// fileSet returns an in-memory folder holding one regular file for each entry of files.
func fileSet(files map[string]string) fstest.MapFS {
	out := fstest.MapFS{}
	for name, content := range files {
		out[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return out
}

// credentialFor returns the credential variable of the source with the ID given. Each source has its own variable, because
// the catalog lets one credential serve one host only, and each test source sits on its own host.
func credentialFor(id string) string {
	return "EXAMPLE_" + strings.ToUpper(strings.ReplaceAll(id, "-", "_")) + "_API_KEY"
}

// outlineSource returns an outline source with the ID and tags given.
func outlineSource(t *testing.T, id string, tags ...string) domain.Source {
	t.Helper()
	return source(t, id, domain.KindOutline, credentialFor(id), tags...)
}

// kiwiSource returns a Kiwi source with the ID and tags given.
func kiwiSource(t *testing.T, id string, tags ...string) domain.Source {
	t.Helper()
	return source(t, id, domain.KindKiwi, credentialFor(id), tags...)
}

// source builds one source of the kind, on the example host of its ID.
func source(t *testing.T, id string, kind domain.SourceKind, credential string, tags ...string) domain.Source {
	t.Helper()
	sid, err := domain.NewSourceID(id)
	if err != nil {
		t.Fatalf("NewSourceID(%q): %v", id, err)
	}
	endpoint, err := domain.NewEndpoint("https://" + id + ".example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef(credential)
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	set, err := domain.NewTagSet(tags...)
	if err != nil {
		t.Fatalf("NewTagSet: %v", err)
	}
	return domain.NewSource(sid, kind, set, endpoint, ref)
}

// catalogOf returns the catalog of the sources given.
func catalogOf(t *testing.T, list ...domain.Source) domain.Catalog {
	t.Helper()
	cat, err := domain.NewCatalog(list)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return cat
}

// rowOf returns the built-in catalog row of the operation, or fails the test when the catalog has none.
func rowOf(t *testing.T, op domain.OperationID) operations.Row {
	t.Helper()
	row, ok := operations.BuiltinCatalog().Lookup(op)
	if !ok {
		t.Fatalf("the built-in catalog has no operation %q", op)
	}
	return row
}

// objectReply returns one object reply that holds every field the operation's row declares, each with the value given.
func objectReply(t *testing.T, op domain.OperationID, value any) domain.Record {
	t.Helper()
	row := rowOf(t, op)
	fields := make([]domain.Field, 0, len(row.Fields))
	for _, name := range row.Fields {
		fields = append(fields, domain.Field{Name: name, Value: value})
	}
	return domain.NewRecord(fields...)
}

// listReply returns one list reply of count items, each holding every field the operation's row declares.
func listReply(t *testing.T, op domain.OperationID, count int) domain.Record {
	t.Helper()
	items := make([]domain.Record, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, objectReply(t, op, "value"))
	}
	return domain.NewList(items)
}

// wiki is a source gateway that counts each call in asked, under its own ID. Each function it is given answers the calls of
// its kind. A nil function means the gateway was not given that answer, which fails the test that reached it.
type wiki struct {
	id    string
	asked *[]string
	read  func(domain.OperationID, domain.Args) (domain.Record, error)
	write func(domain.OperationID, domain.Args) (domain.Record, error)
	fetch func(domain.DocumentID) (domain.SourceDocument, error)
}

func (w *wiki) ask() { *w.asked = append(*w.asked, w.id) }

// ListCollections returns no collections, and counts the call.
func (w *wiki) ListCollections(context.Context) ([]domain.Collection, error) {
	w.ask()
	return nil, nil
}

// CollectionTree returns no nodes, and counts the call.
func (w *wiki) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	w.ask()
	return nil, nil
}

// Document answers with the fetch function, and counts the call.
func (w *wiki) Document(_ context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	w.ask()
	return w.fetch(id)
}

// Read answers with the read function, and counts the call.
func (w *wiki) Read(_ context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	w.ask()
	return w.read(op, args)
}

// Write answers with the write function, and counts the call.
func (w *wiki) Write(_ context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	w.ask()
	return w.write(op, args)
}

// plainGateway serves only the three document calls, so it has no reader and no writer.
type plainGateway struct{ w *wiki }

// ListCollections forwards to the wiki.
func (p plainGateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	return p.w.ListCollections(ctx)
}

// CollectionTree forwards to the wiki.
func (p plainGateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	return p.w.CollectionTree(ctx, id)
}

// Document forwards to the wiki.
func (p plainGateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	return p.w.Document(ctx, id)
}

// sinkRecorder is an export sink that keeps each file it is given. A non-nil fail makes every write return that error.
type sinkRecorder struct {
	written map[string][]byte
	fail    error
}

var _ ports.ExportSink = (*sinkRecorder)(nil)

// Write keeps the bytes under the name, or returns the failure the sink was given.
func (s *sinkRecorder) Write(_ context.Context, name string, data []byte) (string, error) {
	if s.fail != nil {
		return "", s.fail
	}
	if s.written == nil {
		s.written = map[string][]byte{}
	}
	s.written[name] = data
	return "/exports/" + name, nil
}

// builderFor returns a registry builder that hands out the gateway for every source of its kind.
func builderFor(gw ports.SourceGateway) func(domain.Source) (ports.SourceGateway, error) {
	return func(domain.Source) (ports.SourceGateway, error) { return gw, nil }
}
