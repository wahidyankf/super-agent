package sources

import (
	"context"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// TestExportDocumentRefusesAnUnknownSourceBeforeAnyFetch checks that a source ID the catalog does not hold stops the
// export before the gateway is called and before the sink is used.
func TestExportDocumentRefusesAnUnknownSourceBeforeAnyFetch(t *testing.T) {
	cat := kindedCatalog(t, kindedSource{"outline", domain.KindOutline})
	gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "Doc", Markdown: "Body.\n"}}
	svc, _ := kindedService(t, cat, gw, nil, nil)
	sink := newFakeSink()

	_, err := ExportDocument(context.Background(), svc, sink, domain.KindOutline, strPtr("missing"), "doc-1")
	if err == nil {
		t.Fatal("ExportDocument with an unknown source returned no error")
	}
	if len(gw.calls) != 0 || len(sink.names) != 0 {
		t.Fatalf("calls = %v, writes = %v; want none", gw.calls, sink.names)
	}
}
