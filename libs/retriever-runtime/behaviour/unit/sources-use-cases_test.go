package unit_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

// strPtr returns a pointer to s, for the optional source argument of a use case.
func strPtr(s string) *string { return &s }

// wantError fails the test unless err is non-nil and its text is exactly message.
func wantError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", message)
	}
	if err.Error() != message {
		t.Fatalf("error = %q, want %q", err.Error(), message)
	}
}

// outlineService returns a service over the catalog of the sources given, whose outline gateway is gw for every source.
func outlineService(t *testing.T, gw ports.SourceGateway, list ...domain.Source) sources.Service {
	t.Helper()
	return sources.Service{
		Catalog: catalogOf(t, list...),
		Gateways: registry.New(map[domain.SourceKind]registry.Builder{
			domain.KindOutline: builderFor(gw),
		}),
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A read refuses a bad argument before any source is asked
func TestAReadRefusesABadArgumentBeforeAnySourceIsAsked(t *testing.T) {
	cases := []struct {
		name    string
		raw     map[string]any
		message string
	}{
		{"an unknown argument", map[string]any{"bogus": "x"}, "unknown argument"},
		{"no document", map[string]any{}, "a required argument is missing"},
		{"a document ID with a space", map[string]any{"documentId": "doc 1"}, "invalid document ID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var asked []string
			w := &wiki{id: "docs", asked: &asked}
			svc := outlineService(t, w, outlineSource(t, "docs"))
			_, err := sources.Read(context.Background(), svc, domain.OpDocumentsGet, nil, tc.raw)
			wantError(t, err, tc.message)
			if len(asked) != 0 {
				t.Fatalf("source asked %v, want none", asked)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A read with an invalid source ID refuses before any source is asked
func TestAReadWithAnInvalidSourceIDRefusesBeforeAnySourceIsAsked(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	_, err := sources.Read(context.Background(), svc, domain.OpDocumentsGet, strPtr("Outline"),
		map[string]any{"documentId": "doc-1"})
	wantError(t, err, "invalid source ID")
	if len(asked) != 0 {
		t.Fatalf("source asked %v, want none", asked)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A read with a named source asks that source only
func TestAReadWithANamedSourceAsksThatSourceOnly(t *testing.T) {
	var asked []string
	docs := &wiki{id: "docs", asked: &asked}
	wikiGW := &wiki{id: "wiki", asked: &asked, read: func(op domain.OperationID, _ domain.Args) (domain.Record, error) {
		return objectReply(t, op, "doc-1"), nil
	}}
	svc := sources.Service{
		Catalog: catalogOf(t, outlineSource(t, "docs"), outlineSource(t, "wiki")),
		Gateways: registry.New(map[domain.SourceKind]registry.Builder{
			domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
				if src.ID().String() == "wiki" {
					return wikiGW, nil
				}
				return docs, nil
			},
		}),
	}
	got, err := sources.Read(context.Background(), svc, domain.OpDocumentsGet, strPtr("wiki"),
		map[string]any{"documentId": "doc-1"})
	if err != nil {
		t.Fatalf("Read error = %v, want the document", err)
	}
	if len(asked) != 1 || asked[0] != "wiki" {
		t.Fatalf("sources asked = %v, want only wiki", asked)
	}
	if got.Str("id") != "doc-1" {
		t.Fatalf("document id = %q, want doc-1", got.Str("id"))
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A read with no source uses the only source of the kind, or refuses
func TestAReadWithNoSourceUsesTheOnlySourceOfTheKindOrRefuses(t *testing.T) {
	cases := []struct {
		name    string
		list    []domain.Source
		asked   string
		message string
	}{
		{"the only source is named docs", []domain.Source{outlineSource(t, "docs")}, "docs", ""},
		{"two sources are named docs and wiki", []domain.Source{outlineSource(t, "docs"), outlineSource(t, "wiki")}, "",
			"more than one outline source exists: docs, wiki"},
		{"no outline source is configured", []domain.Source{kiwiSource(t, "tests")}, "",
			"no outline source is configured"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var asked []string
			w := &wiki{id: "docs", asked: &asked, read: func(op domain.OperationID, _ domain.Args) (domain.Record, error) {
				return listReply(t, op, 1), nil
			}}
			svc := outlineService(t, w, tc.list...)
			_, err := sources.Read(context.Background(), svc, domain.OpCollectionsList, nil, nil)
			if tc.message != "" {
				wantError(t, err, tc.message)
				if len(asked) != 0 {
					t.Fatalf("source asked %v, want none", asked)
				}
				return
			}
			if err != nil {
				t.Fatalf("Read error = %v, want the source to answer", err)
			}
			if len(asked) != 1 || asked[0] != tc.asked {
				t.Fatalf("sources asked = %v, want %s", asked, tc.asked)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A read reply of the wrong shape returns the unexpected response error
func TestAReadReplyOfTheWrongShapeReturnsTheUnexpectedResponseError(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked, read: func(domain.OperationID, domain.Args) (domain.Record, error) {
		return domain.NewRecord(domain.Field{Name: "unrelated", Value: "x"}), nil
	}}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	got, err := sources.Read(context.Background(), svc, domain.OpDocumentsGet, nil, map[string]any{"documentId": "doc-1"})
	wantError(t, err, "unexpected response from Outline")
	if got.Str("id") != "" || len(got.Fields()) != 0 {
		t.Fatalf("result = %+v, want no result", got)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A write of a read operation is refused
func TestAWriteOfAReadOperationIsRefused(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	_, err := sources.Write(context.Background(), svc, domain.OpDocumentsGet, strPtr("docs"),
		map[string]any{"documentId": "doc-1"})
	wantError(t, err, "the operation is not of this port's effect class")
	if len(asked) != 0 {
		t.Fatalf("source asked %v, want none", asked)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A write with no source is refused and names the source it needs
func TestAWriteWithNoSourceIsRefusedAndNamesTheSourceItNeeds(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	raw := map[string]any{
		"title":        "Notes",
		"collectionId": "550e8400-e29b-41d4-a716-446655440000",
		"text":         "Body",
	}
	_, err := sources.Write(context.Background(), svc, domain.OpDocumentsCreate, nil, raw)
	wantError(t, err, "this operation changes the wiki, so name its source with --source or the source argument")
	if len(asked) != 0 {
		t.Fatalf("source asked %v, want none", asked)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A write reply of the wrong shape reports an unknown outcome
func TestAWriteReplyOfTheWrongShapeReportsAnUnknownOutcome(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked, write: func(domain.OperationID, domain.Args) (domain.Record, error) {
		return domain.NewRecord(domain.Field{Name: "unrelated", Value: "x"}), nil
	}}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	_, err := sources.Write(context.Background(), svc, domain.OpDocumentsCreate, strPtr("docs"), createArgs())
	wantError(t, err, "the change may or may not have happened; check before you retry")
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A write with a named source returns the reply of the writer
func TestAWriteWithANamedSourceReturnsTheReplyOfTheWriter(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked, write: func(op domain.OperationID, _ domain.Args) (domain.Record, error) {
		return objectReply(t, op, "doc-1"), nil
	}}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	got, err := sources.Write(context.Background(), svc, domain.OpDocumentsCreate, strPtr("docs"), createArgs())
	if err != nil {
		t.Fatalf("Write error = %v, want the reply of the writer", err)
	}
	if got.Str("id") != "doc-1" {
		t.Fatalf("reply id = %q, want doc-1", got.Str("id"))
	}
}

// createArgs returns the arguments of a valid outline document create.
func createArgs() map[string]any {
	return map[string]any{
		"title":        "Notes",
		"collectionId": "550e8400-e29b-41d4-a716-446655440000",
		"text":         "Body",
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: An export writes the Markdown under a file name made from the ID
func TestAnExportWritesTheMarkdownUnderAFileNameMadeFromTheID(t *testing.T) {
	cases := []struct{ id, title string }{
		{"doc-1", "Notes"},
		{"doc-2", ""},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			var asked []string
			w := &wiki{id: "docs", asked: &asked, fetch: func(id domain.DocumentID) (domain.SourceDocument, error) {
				return domain.SourceDocument{ID: id.String(), Title: tc.title, Markdown: "# Notes"}, nil
			}}
			svc := outlineService(t, w, outlineSource(t, "docs"))
			sink := &sinkRecorder{}
			location, err := sources.ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, tc.id)
			if err != nil {
				t.Fatalf("ExportDocument error = %v", err)
			}
			if got := string(sink.written[tc.id+".md"]); got != "# Notes" {
				t.Fatalf("file %s.md holds %q, want the body", tc.id, got)
			}
			if location != "/exports/"+tc.id+".md" {
				t.Fatalf("location = %q, want the sink's location", location)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: An export with an invalid document ID refuses before any fetch
func TestAnExportWithAnInvalidDocumentIDRefusesBeforeAnyFetch(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked, fetch: func(domain.DocumentID) (domain.SourceDocument, error) {
		return domain.SourceDocument{}, nil
	}}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	sink := &sinkRecorder{}
	_, err := sources.ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "doc 1")
	wantError(t, err, "invalid document ID")
	if len(sink.written) != 0 || len(asked) != 0 {
		t.Fatalf("written %d files and asked %v, want neither", len(sink.written), asked)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: An export whose fetch fails writes nothing and keeps the fetch error
func TestAnExportWhoseFetchFailsWritesNothingAndKeepsTheFetchError(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked, fetch: func(domain.DocumentID) (domain.SourceDocument, error) {
		return domain.SourceDocument{}, domain.ErrNotFound
	}}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	sink := &sinkRecorder{}
	_, err := sources.ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "doc-1")
	wantError(t, err, "not found")
	if len(sink.written) != 0 {
		t.Fatalf("written %d files, want none", len(sink.written))
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: An export whose file write fails returns the write error
func TestAnExportWhoseFileWriteFailsReturnsTheWriteError(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked, fetch: func(id domain.DocumentID) (domain.SourceDocument, error) {
		return domain.SourceDocument{ID: id.String(), Markdown: "# Notes"}, nil
	}}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	sink := &sinkRecorder{fail: errors.New("disk is full")}
	_, err := sources.ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "doc-1")
	if !errors.Is(err, domain.ErrExportWriteFailed) {
		t.Fatalf("error = %v, want the export write error", err)
	}
	if !strings.HasPrefix(err.Error(), "could not write the exported document") {
		t.Fatalf("error text = %q, want the export write text", err.Error())
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: The source list shows ID, kind, and tags, never the endpoint
func TestTheSourceListShowsIDKindAndTagsNeverTheEndpoint(t *testing.T) {
	cat := catalogOf(t, outlineSource(t, "docs", "wiki"), kiwiSource(t, "tests", "tests"))
	list := sources.ListSources(cat, domain.SourceFilter{})
	if len(list) != 2 {
		t.Fatalf("list holds %d sources, want 2", len(list))
	}
	if list[0].ID != "docs" || list[0].Kind != "outline" || len(list[0].Tags) != 1 || list[0].Tags[0] != "wiki" {
		t.Fatalf("first summary = %+v, want docs, outline, wiki", list[0])
	}
	if list[1].ID != "tests" || list[1].Kind != "kiwi" || len(list[1].Tags) != 1 || list[1].Tags[0] != "tests" {
		t.Fatalf("second summary = %+v, want tests, kiwi, tests", list[1])
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: The source list filters by kind and by every tag given
func TestTheSourceListFiltersByKindAndByEveryTagGiven(t *testing.T) {
	cat := catalogOf(t,
		outlineSource(t, "a", "wiki", "tests"),
		outlineSource(t, "b", "wiki"),
		kiwiSource(t, "c", "wiki", "tests"),
	)
	cases := []struct {
		kind   string
		tags   []string
		listed string
	}{
		{"outline", []string{"wiki"}, "a, b"},
		{"outline", []string{"wiki", "tests"}, "a"},
		{"kiwi", []string{"wiki"}, "c"},
	}
	for _, tc := range cases {
		t.Run(tc.kind+" "+strings.Join(tc.tags, ","), func(t *testing.T) {
			filter, err := domain.NewSourceFilter(strPtr(tc.kind), tc.tags...)
			if err != nil {
				t.Fatalf("NewSourceFilter error = %v", err)
			}
			var ids []string
			for _, s := range sources.ListSources(cat, filter) {
				ids = append(ids, s.ID)
			}
			if got := strings.Join(ids, ", "); got != tc.listed {
				t.Fatalf("listed = %q, want %q", got, tc.listed)
			}
		})
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A named source of another kind is refused for the read
func TestANamedSourceOfAnotherKindIsRefusedForTheRead(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked}
	svc := outlineService(t, w, outlineSource(t, "docs"), kiwiSource(t, "tests"))
	_, err := sources.Read(context.Background(), svc, domain.OpDocumentsGet, strPtr("tests"),
		map[string]any{"documentId": "doc-1"})
	wantError(t, err, "the source is of another kind")
	if len(asked) != 0 {
		t.Fatalf("source asked %v, want none", asked)
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: A composed catalog replaces the built-in rows and runs through its reader
func TestAComposedCatalogReplacesTheBuiltInRowsAndRunsThroughItsReader(t *testing.T) {
	composed, err := operations.NewCatalog(domain.BuiltinKinds(), []operations.Row{{
		ID:     domain.OpKiwiProductsList,
		CLI:    "kiwi products list",
		Effect: domain.EffectRead,
		Fields: []string{"id"},
	}})
	if err != nil {
		t.Fatalf("NewCatalog error = %v", err)
	}
	var asked []string
	kiwi := &wiki{id: "tests", asked: &asked, read: func(domain.OperationID, domain.Args) (domain.Record, error) {
		return domain.NewRecord(domain.Field{Name: "id", Value: "1"}), nil
	}}
	svc := sources.Service{
		Catalog: catalogOf(t, kiwiSource(t, "tests")),
		Gateways: registry.New(map[domain.SourceKind]registry.Builder{
			domain.KindKiwi: builderFor(kiwi),
		}),
		Operations: &composed,
	}
	got, err := sources.Read(context.Background(), svc, domain.OpKiwiProductsList, strPtr("tests"), nil)
	if err != nil {
		t.Fatalf("Read error = %v, want the reader's reply", err)
	}
	if len(asked) != 1 || got.Str("id") != "1" {
		t.Fatalf("asked %v and id %q, want the kiwi reader to answer", asked, got.Str("id"))
	}
}

// @scenario specs/libs/retriever-runtime/behaviours/sources-use-cases.feature: An operation outside the catalog is refused before any reader is asked
func TestAnOperationOutsideTheCatalogIsRefusedBeforeAnyReaderIsAsked(t *testing.T) {
	var asked []string
	w := &wiki{id: "docs", asked: &asked}
	svc := outlineService(t, w, outlineSource(t, "docs"))
	_, err := sources.Read(context.Background(), svc, "documents.vanish", strPtr("docs"), nil)
	wantError(t, err, "unsupported operation")
	if len(asked) != 0 {
		t.Fatalf("reader asked %v, want none", asked)
	}
}
