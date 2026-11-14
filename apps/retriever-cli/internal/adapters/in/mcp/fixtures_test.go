package mcp

import (
	"context"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// fakeGateway answers every source call from fields the test sets. It makes no network call.
type fakeGateway struct {
	collections []domain.Collection
	tree        []domain.TreeNode
	doc         domain.SourceDocument
	err         error
	calls       int
	// reply is the Record that Read returns for the list and account operations, and ops records the operations it was
	// asked to run. The other operations are answered from the typed fields above.
	reply domain.Record
	ops   []domain.OperationID
	// writeReply is the Record that Write returns. writes and writeArgs record each write the gateway was asked to run.
	writeReply domain.Record
	writes     []domain.OperationID
	writeArgs  []domain.Args
}

// Write records the write and its arguments, and answers with the configured reply or error.
func (g *fakeGateway) Write(_ context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	g.calls++
	g.writes = append(g.writes, op)
	g.writeArgs = append(g.writeArgs, args)
	if g.err != nil {
		return domain.Record{}, g.err
	}
	return g.writeReply, nil
}

// Read answers an operation from the configured fields, or with the configured error, and records the operation.
func (g *fakeGateway) Read(_ context.Context, op domain.OperationID, _ domain.Args) (domain.Record, error) {
	g.calls++
	g.ops = append(g.ops, op)
	if g.err != nil {
		return domain.Record{}, g.err
	}
	switch op {
	case domain.OpCollectionsList:
		items := make([]domain.Record, 0, len(g.collections))
		for _, c := range g.collections {
			items = append(items, domain.NewRecord(domain.Field{Name: "id", Value: c.ID}, domain.Field{Name: "name", Value: c.Name}))
		}
		return domain.NewList(items), nil
	case domain.OpCollectionsDocuments:
		items := make([]domain.Record, 0, len(g.tree))
		for _, n := range g.tree {
			var parent any
			if n.ParentID != nil {
				parent = *n.ParentID
			}
			items = append(items, domain.NewRecord(
				domain.Field{Name: "id", Value: n.ID}, domain.Field{Name: "title", Value: n.Title},
				domain.Field{Name: "url", Value: n.URL}, domain.Field{Name: "parentId", Value: parent},
				domain.Field{Name: "depth", Value: n.Depth}))
		}
		return domain.NewList(items), nil
	case domain.OpDocumentsGet:
		return domain.NewRecord(
			domain.Field{Name: "id", Value: g.doc.ID}, domain.Field{Name: "title", Value: g.doc.Title},
			domain.Field{Name: "url", Value: g.doc.URL}, domain.Field{Name: "collectionId", Value: g.doc.CollectionID},
			domain.Field{Name: "revision", Value: 1}, domain.Field{Name: "markdown", Value: g.doc.Markdown}), nil
	case domain.OpCollectionsInfo:
		c := domain.Collection{}
		if len(g.collections) > 0 {
			c = g.collections[0]
		}
		return domain.NewRecord(domain.Field{Name: "id", Value: c.ID}, domain.Field{Name: "name", Value: c.Name}), nil
	}
	return g.reply, nil
}

// ListCollections returns the configured collections or the configured error.
func (g *fakeGateway) ListCollections(context.Context) ([]domain.Collection, error) {
	g.calls++
	return g.collections, g.err
}

// CollectionTree returns the configured nodes or the configured error.
func (g *fakeGateway) CollectionTree(context.Context, domain.CollectionID) ([]domain.TreeNode, error) {
	g.calls++
	return g.tree, g.err
}

// Document returns the configured document or the configured error.
func (g *fakeGateway) Document(context.Context, domain.DocumentID) (domain.SourceDocument, error) {
	g.calls++
	return g.doc, g.err
}

// fakeResolver hands every source the same gateway.
type fakeResolver struct {
	gw ports.SourceGateway
}

// For returns the one gateway the test configured.
func (r fakeResolver) For(domain.Source) (ports.SourceGateway, error) {
	return r.gw, nil
}

// Reader returns the configured gateway as a reader, when it can read operations.
func (r fakeResolver) Reader(domain.Source) (ports.SourceReader, error) {
	reader, ok := r.gw.(ports.SourceReader)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return reader, nil
}

// Writer returns the configured gateway as a writer, when it can write operations.
func (r fakeResolver) Writer(domain.Source) (ports.SourceWriter, error) {
	writer, ok := r.gw.(ports.SourceWriter)
	if !ok {
		return nil, domain.ErrUnsupportedSource
	}
	return writer, nil
}

// mustSource builds a source from valid literals, failing the test on a bad literal.
func mustSource(t *testing.T, id string, tags ...string) domain.Source {
	t.Helper()
	sid, err := domain.NewSourceID(id)
	if err != nil {
		t.Fatal(err)
	}
	kind, err := domain.NewSourceKind("outline")
	if err != nil {
		t.Fatal(err)
	}
	set, err := domain.NewTagSet(tags...)
	if err != nil {
		t.Fatal(err)
	}
	endpoint, err := domain.NewEndpoint("https://docs.example.test")
	if err != nil {
		t.Fatal(err)
	}
	credential, err := domain.NewCredentialRef("EXAMPLE_OUTLINE_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	return domain.NewSource(sid, kind, set, endpoint, credential)
}

// serviceWith returns a service whose catalog holds the given sources and whose gateways all answer from gw.
func serviceWith(t *testing.T, gw ports.SourceGateway, list ...domain.Source) sources.Service {
	t.Helper()
	cat, err := domain.NewCatalog(list)
	if err != nil {
		t.Fatal(err)
	}
	return sources.Service{Catalog: cat, Gateways: fakeResolver{gw: gw}}
}

// emptyService returns a service with one read source and an empty gateway.
func emptyService(t *testing.T) sources.Service {
	t.Helper()
	return serviceWith(t, &fakeGateway{}, mustSource(t, "outline"))
}
