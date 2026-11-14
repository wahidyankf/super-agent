package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestExportDocumentWritesTheMarkdownUnderTheDocumentID checks that an Outline document goes to the sink as its Markdown
// body, under a file name made from its ID. The title does not reach the name.
func TestExportDocumentWritesTheMarkdownUnderTheDocumentID(t *testing.T) {
	cat := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "Getting Started", Markdown: "# Hi\n"}}
	svc, _ := kindedService(t, cat, gw, nil, nil)
	sink := newFakeSink()

	got, err := ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "doc-1")
	if err != nil || got != "doc-1.md" {
		t.Fatalf("ExportDocument = %q, %v; want doc-1.md", got, err)
	}
	if body := string(sink.bodies["doc-1.md"]); body != "# Hi\n" {
		t.Fatalf("written body = %q; want the Markdown of the document", body)
	}
}

// TestExportDocumentNamesAnUntitledOutlineDocumentByID checks that a title with no letters or digits gives the same
// file name as any other title: the document ID.
func TestExportDocumentNamesAnUntitledOutlineDocumentByID(t *testing.T) {
	cat := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "!!!", Markdown: "Body.\n"}}
	svc, _ := kindedService(t, cat, gw, nil, nil)
	sink := newFakeSink()

	got, err := ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "doc-1")
	if err != nil || got != "doc-1.md" {
		t.Fatalf("ExportDocument = %q, %v; want doc-1.md", got, err)
	}
}

// TestExportDocumentRefusesAnInvalidIDBeforeAnyFetch checks that a malformed document ID stops the export before the
// gateway is called and before the sink is used.
func TestExportDocumentRefusesAnInvalidIDBeforeAnyFetch(t *testing.T) {
	cat := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	gw := &fakeGateway{}
	svc, _ := kindedService(t, cat, gw, nil, nil)
	sink := newFakeSink()

	_, err := ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "bad/id")
	if !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("error = %v; want ErrInvalidDocumentID", err)
	}
	if len(gw.calls) != 0 || len(sink.names) != 0 {
		t.Fatalf("calls = %v, writes = %v; want none", gw.calls, sink.names)
	}
}

// TestExportDocumentStopsWhenTheFetchFails checks that a failed fetch writes nothing and keeps the gateway's error.
func TestExportDocumentStopsWhenTheFetchFails(t *testing.T) {
	cat := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	gw := &fakeGateway{err: domain.ErrDocumentNotFound}
	svc, _ := kindedService(t, cat, gw, nil, nil)
	sink := newFakeSink()

	_, err := ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "doc-1")
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("error = %v; want ErrDocumentNotFound", err)
	}
	if len(sink.names) != 0 {
		t.Fatalf("writes = %v; want none", sink.names)
	}
}

// TestExportDocumentReturnsTheSinkError checks that a failed write keeps the sink's error, so the caller can classify it.
func TestExportDocumentReturnsTheSinkError(t *testing.T) {
	cat := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "Doc", Markdown: "Body.\n"}}
	svc, _ := kindedService(t, cat, gw, nil, nil)
	sink := newFakeSink()
	sink.writeErr = domain.ErrExportWriteFailed

	_, err := ExportDocument(context.Background(), svc, sink, domain.KindOutline, nil, "doc-1")
	if !errors.Is(err, domain.ErrExportWriteFailed) {
		t.Fatalf("error = %v; want ErrExportWriteFailed", err)
	}
}
