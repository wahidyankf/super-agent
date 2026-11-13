package sources

import (
	"context"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// fakeGateway is an in-memory ports.SourceGateway. It returns the configured
// values and records every call as "collections", "tree:<id>", or "doc:<id>".
type fakeGateway struct {
	collections []domain.Collection
	tree        []domain.TreeNode
	doc         domain.SourceDocument
	err         error
	calls       []string
}

func (g *fakeGateway) ListCollections(context.Context) ([]domain.Collection, error) {
	g.calls = append(g.calls, "collections")
	return g.collections, g.err
}

func (g *fakeGateway) CollectionTree(_ context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	g.calls = append(g.calls, "tree:"+id.String())
	return g.tree, g.err
}

func (g *fakeGateway) Document(_ context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	g.calls = append(g.calls, "doc:"+id.String())
	return g.doc, g.err
}

// fakeReader is an in-memory ports.SourceReader. It returns the configured reply and records every operation it
// was asked to run.
type fakeReader struct {
	reply domain.Record
	err   error
	ops   []domain.OperationID
	args  []domain.Args
	// byOp, when set, answers the operations it names with their own reply, and every other operation with reply.
	byOp map[domain.OperationID]domain.Record
}

func (r *fakeReader) Read(_ context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	r.ops = append(r.ops, op)
	r.args = append(r.args, args)
	if rec, ok := r.byOp[op]; ok {
		return rec, r.err
	}
	return r.reply, r.err
}

// fakeWriter is an in-memory ports.SourceWriter. It returns the configured reply and records every operation it
// was asked to run, so a test can prove that a refused or dry-run call never reached it.
type fakeWriter struct {
	reply domain.Record
	err   error
	ops   []domain.OperationID
	args  []domain.Args
}

func (w *fakeWriter) Write(_ context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	w.ops = append(w.ops, op)
	w.args = append(w.args, args)
	return w.reply, w.err
}

// fakeAdmin is an in-memory ports.SourceAdmin. It records each destructive or admin call it receives.
type fakeAdmin struct {
	reply domain.Record
	err   error
	ops   []domain.OperationID
	args  []domain.Args
}

func (a *fakeAdmin) Administer(_ context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	a.ops = append(a.ops, op)
	a.args = append(a.args, args)
	return a.reply, a.err
}

// fakeResolver is an in-memory ports.GatewayResolver. It serves one gateway per source ID and records each
// source it was asked for, so a test can prove that a bad input never reached For. reader is the reader every
// source gets, and writer is the writer; nil means the source has no reader or no writer.
type fakeResolver struct {
	gateways map[string]ports.SourceGateway
	asked    []string
	reader   *fakeReader
	writer   *fakeWriter
	admin    *fakeAdmin
}

func (r *fakeResolver) For(src domain.Source) (ports.SourceGateway, error) {
	r.asked = append(r.asked, src.ID().String())
	gw, ok := r.gateways[src.ID().String()]
	if !ok || gw == nil {
		return nil, domain.ErrUnsupportedSource
	}
	return gw, nil
}

func (r *fakeResolver) Reader(src domain.Source) (ports.SourceReader, error) {
	if _, err := r.For(src); err != nil {
		return nil, err
	}
	if r.reader == nil {
		return nil, domain.ErrUnsupportedSource
	}
	return r.reader, nil
}

func (r *fakeResolver) Writer(src domain.Source) (ports.SourceWriter, error) {
	if _, err := r.For(src); err != nil {
		return nil, err
	}
	if r.writer == nil {
		return nil, domain.ErrUnsupportedSource
	}
	return r.writer, nil
}

func (r *fakeResolver) Admin(src domain.Source) (ports.SourceAdmin, error) {
	if _, err := r.For(src); err != nil {
		return nil, err
	}
	if r.admin == nil {
		return nil, domain.ErrUnsupportedSource
	}
	return r.admin, nil
}

// fakeSink is an in-memory ports.ExportSink. writeErr makes Write fail. names records the file name of every successful
// write, in order, and bodies holds the bytes written under each name.
type fakeSink struct {
	writeErr error
	names    []string
	bodies   map[string][]byte
}

func newFakeSink() *fakeSink {
	return &fakeSink{bodies: map[string][]byte{}}
}

func (s *fakeSink) Write(_ context.Context, name string, data []byte) (string, error) {
	if s.writeErr != nil {
		return "", s.writeErr
	}
	s.names = append(s.names, name)
	s.bodies[name] = data
	return name, nil
}

// testCatalog builds a catalog of outline sources with the given IDs. The first ID is read-only, and the rest
// get distinct hosts and keys, so the catalog invariants hold.
func testCatalog(t *testing.T, ids ...string) domain.Catalog {
	t.Helper()
	var list []domain.Source
	for i, raw := range ids {
		id, err := domain.NewSourceID(raw)
		if err != nil {
			t.Fatalf("NewSourceID(%q): %v", raw, err)
		}
		ep, err := domain.NewEndpoint(fmt.Sprintf("https://wiki%d.example.com", i))
		if err != nil {
			t.Fatalf("NewEndpoint: %v", err)
		}
		ref, err := domain.NewCredentialRef(fmt.Sprintf("KEY%d_API_KEY", i))
		if err != nil {
			t.Fatalf("NewCredentialRef: %v", err)
		}
		tags, _ := domain.NewTagSet()
		list = append(list, domain.NewSource(id, domain.KindOutline, tags, domain.AccessRead, ep, ref))
	}
	cat, err := domain.NewCatalog(list)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return cat
}

// serviceOf returns a Service over the catalog of ids, with one gateway per ID taken from gws.
func serviceOf(t *testing.T, gws map[string]ports.SourceGateway) (Service, *fakeResolver) {
	t.Helper()
	ids := make([]string, 0, len(gws))
	for id := range gws {
		ids = append(ids, id)
	}
	resolver := &fakeResolver{gateways: gws}
	return Service{Catalog: testCatalog(t, ids...), Gateways: resolver}, resolver
}

// outlineService returns a Service with one outline source named "outline", served by gw.
func outlineService(t *testing.T, gw ports.SourceGateway) (Service, *fakeResolver) {
	t.Helper()
	return serviceOf(t, map[string]ports.SourceGateway{"outline": gw})
}

// strPtr returns a pointer to s, for the optional source argument.
func strPtr(s string) *string {
	return &s
}

// mustTagSet returns the tag set of tags, failing the test on an invalid tag.
func mustTagSet(t *testing.T, tags ...string) domain.TagSet {
	t.Helper()
	set, err := domain.NewTagSet(tags...)
	if err != nil {
		t.Fatalf("NewTagSet(%v): %v", tags, err)
	}
	return set
}
