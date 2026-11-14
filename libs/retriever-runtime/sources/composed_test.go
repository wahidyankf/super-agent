package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// specReader is a reader that records the operations it is asked for and returns one fixed list.
type specReader struct {
	calls []domain.OperationID
	reply domain.Record
}

// Read records the operation and returns the fixed reply.
func (r *specReader) Read(_ context.Context, op domain.OperationID, _ domain.Args) (domain.Record, error) {
	r.calls = append(r.calls, op)
	return r.reply, nil
}

// specResolver returns the same reader for every source, and no writer.
type specResolver struct{ reader *specReader }

// For has no full gateway, so the reads of a composed row never use it.
func (r specResolver) For(domain.Source) (ports.SourceGateway, error) {
	return nil, domain.ErrUnsupportedSource
}

// Reader returns the fixed reader.
func (r specResolver) Reader(domain.Source) (ports.SourceReader, error) {
	return r.reader, nil
}

// Writer has no writer in these tests.
func (r specResolver) Writer(domain.Source) (ports.SourceWriter, error) {
	return nil, domain.ErrUnsupportedSource
}

// composedService returns a service whose operation catalog holds one spec kind, example-wiki, with one read row, and one
// source of that kind named wiki.
func composedService(t *testing.T) (Service, *specReader) {
	t.Helper()
	kinds, err := domain.NewKindSet(domain.KindDefinition{Name: "example-wiki"})
	if err != nil {
		t.Fatalf("NewKindSet: %v", err)
	}
	kind, err := kinds.Kind("example-wiki")
	if err != nil {
		t.Fatalf("Kind: %v", err)
	}
	cat, err := operations.NewCatalog(kinds, []operations.Row{{
		ID:      "example-wiki.docs.list",
		CLI:     "example-wiki docs list",
		Effect:  domain.EffectRead,
		ListKey: "items",
		Fields:  []string{"id"},
	}})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	id, err := domain.NewSourceID("wiki")
	if err != nil {
		t.Fatalf("NewSourceID: %v", err)
	}
	ep, err := domain.NewEndpoint("https://wiki.example.test")
	if err != nil {
		t.Fatalf("NewEndpoint: %v", err)
	}
	ref, err := domain.NewCredentialRef("EXAMPLE_WIKI_API_KEY")
	if err != nil {
		t.Fatalf("NewCredentialRef: %v", err)
	}
	tags, _ := domain.NewTagSet()
	catalog, err := domain.NewCatalog([]domain.Source{domain.NewSource(id, kind, tags, ep, ref)})
	if err != nil {
		t.Fatalf("NewCatalog sources: %v", err)
	}
	reader := &specReader{reply: domain.NewList([]domain.Record{domain.NewRecord(domain.Field{Name: "id", Value: "d1"})})}
	return Service{Catalog: catalog, Gateways: specResolver{reader: reader}, Operations: &cat}, reader
}

// TestAComposedRowRunsThroughTheReaderOfItsKind checks that a row the composed catalog holds, and that the built-in
// catalog does not, reaches the reader of its source.
func TestAComposedRowRunsThroughTheReaderOfItsKind(t *testing.T) {
	svc, reader := composedService(t)
	rec, err := Read(context.Background(), svc, "example-wiki.docs.list", nil, nil)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(reader.calls) != 1 || reader.calls[0] != "example-wiki.docs.list" {
		t.Fatalf("reader calls = %v, want one call of example-wiki.docs.list", reader.calls)
	}
	if len(rec.Items()) != 1 {
		t.Fatalf("items = %d, want 1", len(rec.Items()))
	}
}

// TestTheComposedCatalogReplacesTheBuiltInRows checks that once a composed catalog is set, a built-in operation that it
// does not hold is refused before any reader is asked.
func TestTheComposedCatalogReplacesTheBuiltInRows(t *testing.T) {
	svc, reader := composedService(t)
	_, err := Read(context.Background(), svc, domain.OpCollectionsDocuments, nil, nil)
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation", err)
	}
	if len(reader.calls) != 0 {
		t.Fatalf("reader was asked %d times, want 0", len(reader.calls))
	}
}

// TestWithoutAComposedCatalogTheBuiltInRowsStillRun checks the default: a service with no composed catalog reads the
// built-in rows, as it did before the composed catalog existed.
func TestWithoutAComposedCatalogTheBuiltInRowsStillRun(t *testing.T) {
	svc, reader := composedService(t)
	svc.Operations = nil
	_, err := Read(context.Background(), svc, "example-wiki.docs.list", nil, nil)
	if !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("err = %v, want ErrUnsupportedOperation for a row only the composed catalog holds", err)
	}
	if len(reader.calls) != 0 {
		t.Fatalf("reader was asked %d times, want 0", len(reader.calls))
	}
}
