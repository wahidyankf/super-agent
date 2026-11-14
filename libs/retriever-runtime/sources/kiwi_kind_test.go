package sources

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// kindedSource describes one source of a test catalog: its ID and kind.
type kindedSource struct {
	id   string
	kind domain.SourceKind
}

// kindedCatalog returns a catalog of the given sources. Each source gets its own host and key name, so the catalog
// invariants hold.
func kindedCatalog(t *testing.T, list ...kindedSource) domain.Catalog {
	t.Helper()
	var out []domain.Source
	for i, s := range list {
		id, err := domain.NewSourceID(s.id)
		if err != nil {
			t.Fatalf("NewSourceID(%q): %v", s.id, err)
		}
		ep, err := domain.NewEndpoint(fmt.Sprintf("https://%s.example.com", s.id))
		if err != nil {
			t.Fatalf("NewEndpoint: %v", err)
		}
		ref, err := domain.NewCredentialRef(fmt.Sprintf("KEY%d_API_KEY", i))
		if err != nil {
			t.Fatalf("NewCredentialRef: %v", err)
		}
		out = append(out, domain.NewSource(id, s.kind, mustTagSet(t), ep, ref))
	}
	cat, err := domain.NewCatalog(out)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	return cat
}

// kindedService serves every source of the catalog through one gateway, reader, and writer.
func kindedService(t *testing.T, cat domain.Catalog, gw ports.SourceGateway, reader *fakeReader, writer *fakeWriter) (Service, *fakeResolver) {
	t.Helper()
	gws := map[string]ports.SourceGateway{}
	for _, src := range cat.All() {
		gws[src.ID().String()] = gw
	}
	resolver := &fakeResolver{gateways: gws, reader: reader, writer: writer}
	return Service{Catalog: cat, Gateways: resolver}, resolver
}

// kiwiDocumentReply is a Kiwi test case as kiwi.documents.get answers it.
func kiwiDocumentReply() domain.Record {
	return domain.NewRecord(
		domain.Field{Name: "id", Value: "case-12"},
		domain.Field{Name: "title", Value: "Login works"},
		domain.Field{Name: "url", Value: "https://kiwi.example.com/case/12/"},
		domain.Field{Name: "markdown", Value: "# Login works\n"},
		domain.Field{Name: "collectionId", Value: "product-3"},
	)
}

// TestSelectSourceIsKindAware checks that every caller's kind picks a source of that kind, and that a source ID of another
// kind is refused before any gateway call.
func TestSelectSourceIsKindAware(t *testing.T) {
	cat := kindedCatalog(t,
		kindedSource{"outline", domain.KindOutline},
		kindedSource{"kiwi", domain.KindKiwi},
	)
	svc, resolver := kindedService(t, cat, &fakeGateway{}, nil, nil)

	got, err := Resolve(svc, domain.KindKiwi, nil)
	if err != nil || got.ID().String() != "kiwi" {
		t.Fatalf("Resolve(kiwi, nil) = %v, %v; want the kiwi source", got.ID(), err)
	}
	got, err = Resolve(svc, domain.KindOutline, nil)
	if err != nil || got.ID().String() != "outline" {
		t.Fatalf("Resolve(outline, nil) = %v, %v; want the outline source", got.ID(), err)
	}
	if _, err := Resolve(svc, domain.KindKiwi, strPtr("outline")); !errors.Is(err, domain.ErrWrongSourceKind) {
		t.Fatalf("Resolve(kiwi, outline) error = %v, want ErrWrongSourceKind", err)
	}
	if _, err := Resolve(svc, domain.KindOutline, strPtr("kiwi")); !errors.Is(err, domain.ErrWrongSourceKind) {
		t.Fatalf("Resolve(outline, kiwi) error = %v, want ErrWrongSourceKind", err)
	}
	if _, err := Resolve(svc, domain.KindKiwi, strPtr("Kiwi")); !errors.Is(err, domain.ErrInvalidSourceID) {
		t.Fatalf("Resolve(kiwi, Kiwi) error = %v, want ErrInvalidSourceID", err)
	}
	if len(resolver.asked) != 0 {
		t.Fatalf("Resolve reached the gateway resolver: %v", resolver.asked)
	}
}

// TestDefaultRulePerKind checks the default rule for each kind: exactly one source of the kind, or the typed error that
// names the kind.
func TestDefaultRulePerKind(t *testing.T) {
	two := kindedCatalog(t,
		kindedSource{"outline", domain.KindOutline},
		kindedSource{"kiwi-a", domain.KindKiwi},
		kindedSource{"kiwi-b", domain.KindKiwi},
	)
	svc, _ := kindedService(t, two, &fakeGateway{}, nil, nil)
	if got, err := Resolve(svc, domain.KindOutline, nil); err != nil || got.ID().String() != "outline" {
		t.Fatalf("Resolve(outline, nil) with two kiwi sources = %v, %v", got.ID(), err)
	}
	_, err := Resolve(svc, domain.KindKiwi, nil)
	var amb domain.AmbiguousSourceError
	if !errors.As(err, &amb) || amb.Kind != domain.KindKiwi || len(amb.IDs) != 2 {
		t.Fatalf("Resolve(kiwi, nil) with two kiwi sources error = %v, want AmbiguousSourceError for kiwi", err)
	}

	outlineOnly := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	svc, _ = kindedService(t, outlineOnly, &fakeGateway{}, nil, nil)
	_, err = Resolve(svc, domain.KindKiwi, nil)
	var none domain.NoSourceOfKindError
	if !errors.As(err, &none) || none.Kind != domain.KindKiwi || err.Error() != "no kiwi source is configured" {
		t.Fatalf("Resolve(kiwi, nil) with no kiwi source error = %v, want NoSourceOfKindError for kiwi", err)
	}
	if _, err := Resolve(svc, domain.KindOutline, nil); err != nil {
		t.Fatalf("Resolve(outline, nil) = %v", err)
	}
}

// TestListGetAndExportTakeTheKind checks that the list, tree, and document use cases select by the kind they are given,
// and reach the gateway of that source only.
func TestListGetAndExportTakeTheKind(t *testing.T) {
	cat := kindedCatalog(t,
		kindedSource{"outline", domain.KindOutline},
		kindedSource{"kiwi", domain.KindKiwi},
	)
	gw := &fakeGateway{
		collections: []domain.Collection{{ID: "product-3", Name: "Lending"}},
		tree:        []domain.TreeNode{{ID: "plan-1"}},
		doc:         domain.SourceDocument{ID: "case-12", Title: "Login works", Markdown: "x"},
	}
	svc, resolver := kindedService(t, cat, gw, nil, nil)

	if got, err := ListCollections(context.Background(), svc, domain.KindKiwi, nil); err != nil || len(got) != 1 {
		t.Fatalf("ListCollections(kiwi) = %v, %v", got, err)
	}
	if _, err := ListCollectionTree(context.Background(), svc, domain.KindKiwi, strPtr("kiwi"), "product-3"); err != nil {
		t.Fatalf("ListCollectionTree(kiwi) error = %v", err)
	}
	if _, err := GetDocument(context.Background(), svc, domain.KindKiwi, nil, "case-12"); err != nil {
		t.Fatalf("GetDocument(kiwi) error = %v", err)
	}
	if len(resolver.asked) != 3 || resolver.asked[0] != "kiwi" {
		t.Fatalf("the resolver was asked for %v, want three calls for kiwi", resolver.asked)
	}
	if _, err := ListCollections(context.Background(), svc, domain.KindKiwi, strPtr("outline")); !errors.Is(err, domain.ErrWrongSourceKind) {
		t.Fatalf("ListCollections(kiwi, outline) error = %v, want ErrWrongSourceKind", err)
	}
}

// TestReadTakesTheKindOfTheRow checks that a Kiwi row runs on a Kiwi source, and that the Outline catalog is not searched
// for it.
func TestReadTakesTheKindOfTheRow(t *testing.T) {
	cat := kindedCatalog(t,
		kindedSource{"outline", domain.KindOutline},
		kindedSource{"kiwi", domain.KindKiwi},
	)
	reader := &fakeReader{reply: kiwiDocumentReply()}
	svc, resolver := kindedService(t, cat, &fakeGateway{}, reader, nil)

	rec, err := Read(context.Background(), svc, domain.OpKiwiDocumentsGet, nil, map[string]any{"documentId": "case-12"})
	if err != nil {
		t.Fatalf("Read(kiwi.documents.get) error = %v", err)
	}
	if got := rec.Str("title"); got != "Login works" {
		t.Fatalf("title = %q, want Login works", got)
	}
	if len(reader.ops) != 1 || reader.ops[0] != domain.OpKiwiDocumentsGet || len(resolver.asked) != 1 || resolver.asked[0] != "kiwi" {
		t.Fatalf("reader ops %v, resolver asked %v; want the kiwi read only", reader.ops, resolver.asked)
	}
	if _, err := Read(context.Background(), svc, domain.OpKiwiDocumentsGet, strPtr("outline"), map[string]any{"documentId": "case-12"}); !errors.Is(err, domain.ErrWrongSourceKind) {
		t.Fatalf("Read(kiwi row, outline) error = %v, want ErrWrongSourceKind", err)
	}

	outlineOnly := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	svc, _ = kindedService(t, outlineOnly, &fakeGateway{}, reader, nil)
	_, err = Read(context.Background(), svc, domain.OpKiwiProductsList, nil, nil)
	var none domain.NoSourceOfKindError
	if !errors.As(err, &none) || none.Kind != domain.KindKiwi {
		t.Fatalf("Read(kiwi row) with no kiwi source error = %v, want NoSourceOfKindError for kiwi", err)
	}
}

// TestRowOfRefusesAnUnknownOperation checks the guard that every use case runs first: an operation the catalog does not
// hold is ErrUnsupportedOperation, and a known operation is returned with its kind.
func TestRowOfRefusesAnUnknownOperation(t *testing.T) {
	if _, _, err := rowOf(Service{}, domain.OperationID("nope.nope")); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("rowOf(unknown) error = %v, want ErrUnsupportedOperation", err)
	}
	row, kind, err := rowOf(Service{}, domain.OpKiwiPlansAddCase)
	if err != nil || kind != domain.KindKiwi || row.ID != domain.OpKiwiPlansAddCase {
		t.Fatalf("rowOf(kiwi.plans.add-case) = %v, %v, %v", row.ID, kind, err)
	}
}

// TestExportFileNameIsTheDocumentID checks that every kind names its export file by the document ID alone. The title never
// decides the name, so a second export of one document replaces the first, and two documents never share a file.
func TestExportFileNameIsTheDocumentID(t *testing.T) {
	cases := []struct {
		name  string
		kind  domain.SourceKind
		title string
		id    string
		want  string
	}{
		{"outline names the file by its ID", domain.KindOutline, "Getting Started", "doc-1", "doc-1.md"},
		{"outline ignores a title with no slug", domain.KindOutline, "!!!", "doc-1", "doc-1.md"},
		{"kiwi names the file by its case ID", domain.KindKiwi, "Login works", "case-12", "case-12.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cat := kindedCatalog(t, kindedSource{"src", tc.kind})
			gw := &fakeGateway{doc: domain.SourceDocument{ID: tc.id, Title: tc.title, Markdown: "Body.\n"}}
			svc, _ := kindedService(t, cat, gw, nil, nil)
			sink := newFakeSink()
			got, err := ExportDocument(context.Background(), svc, sink, tc.kind, nil, tc.id)
			if err != nil || got != tc.want {
				t.Fatalf("ExportDocument(%s) = %q, %v; want %s", tc.kind, got, err, tc.want)
			}
			if _, ok := sink.bodies[tc.want]; !ok {
				t.Fatalf("the file %s was not written", tc.want)
			}
		})
	}
}

// TestExportDocumentNamesAKiwiCaseByID checks the whole export: the file of a Kiwi case is named by its ID, whatever its
// title.
func TestExportDocumentNamesAKiwiCaseByID(t *testing.T) {
	cat := kindedCatalog(t, kindedSource{"kiwi", domain.KindKiwi})
	gw := &fakeGateway{doc: domain.SourceDocument{ID: "case-12", Title: "Login works", Markdown: "Body.\n"}}
	svc, _ := kindedService(t, cat, gw, nil, nil)
	sink := newFakeSink()
	got, err := ExportDocument(context.Background(), svc, sink, domain.KindKiwi, nil, "case-12")
	const want = "case-12.md"
	if err != nil || got != want {
		t.Fatalf("ExportDocument(kiwi) = %q, %v; want %s", got, err, want)
	}
	if _, ok := sink.bodies[want]; !ok {
		t.Fatalf("the file %s was not written", want)
	}
}
