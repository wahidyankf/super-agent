package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

func TestListCollectionsUsesTheSelectedSource(t *testing.T) {
	gw := &fakeGateway{collections: []domain.Collection{{ID: "col-1", Name: "Team"}}}
	svc, resolver := outlineService(t, gw)
	got, err := ListCollections(context.Background(), svc, domain.KindOutline, strPtr("outline"))
	if err != nil || len(got) != 1 || got[0].ID != "col-1" {
		t.Fatalf("ListCollections = %v, %v", got, err)
	}
	if len(resolver.asked) != 1 || resolver.asked[0] != "outline" {
		t.Fatalf("resolver was asked for %v, want [outline]", resolver.asked)
	}
}

func TestListCollectionsWithNoSourceUsesTheOnlyOutlineSource(t *testing.T) {
	gw := &fakeGateway{collections: []domain.Collection{{ID: "col-1", Name: "Team"}}}
	svc, _ := outlineService(t, gw)
	if got, err := ListCollections(context.Background(), svc, domain.KindOutline, nil); err != nil || len(got) != 1 {
		t.Fatalf("ListCollections(nil) = %v, %v", got, err)
	}
}

func TestSourceErrorsStopBeforeAnyGatewayCall(t *testing.T) {
	cases := []struct {
		name    string
		source  *string
		ids     []string
		wantErr error
	}{
		{name: "an invalid source ID", source: strPtr("Outline"), ids: []string{"outline"}, wantErr: domain.ErrInvalidSourceID},
		{name: "an unknown source ID", source: strPtr("rnd"), ids: []string{"outline"}, wantErr: domain.ErrUnknownSource},
		{name: "an omitted source with two outline sources", source: nil, ids: []string{"outline", "outline-engineering"}, wantErr: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeGateway{}
			gws := map[string]ports.SourceGateway{}
			for _, id := range tc.ids {
				gws[id] = gw
			}
			svc, resolver := serviceOf(t, gws)
			_, err := ListCollections(context.Background(), svc, domain.KindOutline, tc.source)
			if err == nil || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
				t.Fatalf("ListCollections error = %v, want %v", err, tc.wantErr)
			}
			var amb domain.AmbiguousSourceError
			if tc.source == nil && !errors.As(err, &amb) {
				t.Fatalf("omitted source error = %v, want AmbiguousSourceError", err)
			}
			if len(resolver.asked) != 0 || len(gw.calls) != 0 {
				t.Fatalf("a bad source reached the resolver %v or the gateway %v", resolver.asked, gw.calls)
			}
		})
	}
}

func TestListCollectionTreeValidatesTheCollectionBeforeTheGateway(t *testing.T) {
	cases := []struct {
		name      string
		id        string
		wantErr   error
		wantCalls []string
	}{
		{name: "a dotted collection ID is invalid and sends no request", id: "col.1", wantErr: domain.ErrInvalidCollectionID},
		{name: "an empty collection ID is invalid and sends no request", id: "", wantErr: domain.ErrInvalidCollectionID},
		{name: "a valid collection ID reaches the gateway once", id: "col-1", wantCalls: []string{"tree:col-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeGateway{tree: []domain.TreeNode{{ID: "doc-1", Title: "Root"}}}
			svc, resolver := outlineService(t, gw)
			got, err := ListCollectionTree(context.Background(), svc, domain.KindOutline, strPtr("outline"), tc.id)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || len(gw.calls) != 0 || len(resolver.asked) != 0 {
					t.Fatalf("error = %v, calls = %v, asked = %v", err, gw.calls, resolver.asked)
				}
				return
			}
			if err != nil || len(got) != 1 || len(gw.calls) != 1 || gw.calls[0] != tc.wantCalls[0] {
				t.Fatalf("tree = %v, err = %v, calls = %v", got, err, gw.calls)
			}
		})
	}
}

func TestListCollectionTreePassesGatewayErrors(t *testing.T) {
	svc, _ := outlineService(t, &fakeGateway{err: domain.ErrCollectionNotFound})
	if _, err := ListCollectionTree(context.Background(), svc, domain.KindOutline, strPtr("outline"), "col-missing"); !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("error = %v, want ErrCollectionNotFound", err)
	}
}

func TestGetDocumentValidatesTheDocumentBeforeTheGateway(t *testing.T) {
	cases := []struct {
		name      string
		id        string
		wantErr   error
		wantCalls []string
	}{
		{name: "a dotted document ID is invalid and sends no request", id: "doc.1", wantErr: domain.ErrInvalidDocumentID},
		{name: "a semicolon in the document ID is invalid", id: "doc;ls", wantErr: domain.ErrInvalidDocumentID},
		{name: "a valid document ID reaches the gateway once", id: "doc-1", wantCalls: []string{"doc:doc-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "Getting Started"}}
			svc, _ := outlineService(t, gw)
			doc, err := GetDocument(context.Background(), svc, domain.KindOutline, strPtr("outline"), tc.id)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || len(gw.calls) != 0 {
					t.Fatalf("error = %v, calls = %v", err, gw.calls)
				}
				return
			}
			if err != nil || doc.Title != "Getting Started" || len(gw.calls) != 1 || gw.calls[0] != tc.wantCalls[0] {
				t.Fatalf("doc = %+v, err = %v, calls = %v", doc, err, gw.calls)
			}
		})
	}
}

func TestGetDocumentMissingReturnsNotFound(t *testing.T) {
	svc, _ := outlineService(t, &fakeGateway{err: domain.ErrDocumentNotFound})
	if _, err := GetDocument(context.Background(), svc, domain.KindOutline, strPtr("outline"), "doc-missing"); !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("error = %v, want ErrDocumentNotFound", err)
	}
}

// badSourceCase is a source argument that must stop a use case before any gateway call.
type badSourceCase struct {
	name          string
	source        *string
	ids           []string
	wantErr       error
	wantAmbiguous bool
}

// badSourceCases are the arguments every source-scoped use case must refuse with zero gateway calls: an invalid ID,
// an unknown ID, and an omitted source when two outline sources exist.
func badSourceCases() []badSourceCase {
	return []badSourceCase{
		{name: "an invalid source ID", source: strPtr("Outline"), ids: []string{"outline"}, wantErr: domain.ErrInvalidSourceID},
		{name: "an unknown source ID", source: strPtr("rnd"), ids: []string{"outline"}, wantErr: domain.ErrUnknownSource},
		{name: "an omitted source with two outline sources", source: nil, ids: []string{"outline", "outline-engineering"}, wantAmbiguous: true},
	}
}

// gatewaysFor maps every ID to gw, so each source of the catalog is served by the same recording gateway.
func gatewaysFor(gw ports.SourceGateway, ids []string) map[string]ports.SourceGateway {
	gws := map[string]ports.SourceGateway{}
	for _, id := range ids {
		gws[id] = gw
	}
	return gws
}

// checkBadSourceError fails the test unless err is the case's refusal.
func checkBadSourceError(t *testing.T, tc badSourceCase, err error) {
	t.Helper()
	var amb domain.AmbiguousSourceError
	if tc.wantAmbiguous {
		if !errors.As(err, &amb) {
			t.Fatalf("error = %v, want AmbiguousSourceError", err)
		}
		return
	}
	if !errors.Is(err, tc.wantErr) {
		t.Fatalf("error = %v, want %v", err, tc.wantErr)
	}
}

func TestListCollectionTreeStopsOnABadSourceBeforeTheGateway(t *testing.T) {
	for _, tc := range badSourceCases() {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeGateway{tree: []domain.TreeNode{{ID: "doc-1", Title: "Root"}}}
			svc, resolver := serviceOf(t, gatewaysFor(gw, tc.ids))
			_, err := ListCollectionTree(context.Background(), svc, domain.KindOutline, tc.source, "col-1")
			checkBadSourceError(t, tc, err)
			if len(resolver.asked) != 0 || len(gw.calls) != 0 {
				t.Fatalf("a bad source reached the resolver %v or the gateway %v", resolver.asked, gw.calls)
			}
		})
	}
}

func TestGetDocumentStopsOnABadSourceBeforeTheGateway(t *testing.T) {
	for _, tc := range badSourceCases() {
		t.Run(tc.name, func(t *testing.T) {
			gw := &fakeGateway{doc: domain.SourceDocument{ID: "doc-1", Title: "Getting Started"}}
			svc, resolver := serviceOf(t, gatewaysFor(gw, tc.ids))
			_, err := GetDocument(context.Background(), svc, domain.KindOutline, tc.source, "doc-1")
			checkBadSourceError(t, tc, err)
			if len(resolver.asked) != 0 || len(gw.calls) != 0 {
				t.Fatalf("a bad source reached the resolver %v or the gateway %v", resolver.asked, gw.calls)
			}
		})
	}
}

func TestNilGatewayIsUnsupported(t *testing.T) {
	svc, _ := serviceOf(t, map[string]ports.SourceGateway{"outline": nil})
	if _, err := ListCollections(context.Background(), svc, domain.KindOutline, strPtr("outline")); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("error = %v, want ErrUnsupportedSource", err)
	}
}

// TestEveryUseCaseStopsWhenTheResolverHasNoGateway checks that each use case that needs a gateway returns the
// resolver error, and that an export writes nothing when the gateway is missing.
func TestEveryUseCaseStopsWhenTheResolverHasNoGateway(t *testing.T) {
	ctx := context.Background()
	svc, _ := serviceOf(t, map[string]ports.SourceGateway{"outline": nil})
	sink := newFakeSink()

	if _, err := ListCollectionTree(ctx, svc, domain.KindOutline, strPtr("outline"), "col-1"); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("ListCollectionTree error = %v, want ErrUnsupportedSource", err)
	}
	if _, err := GetDocument(ctx, svc, domain.KindOutline, strPtr("outline"), "doc-1"); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("GetDocument error = %v, want ErrUnsupportedSource", err)
	}
	if _, err := ExportDocument(ctx, svc, sink, domain.KindOutline, strPtr("outline"), "doc-1"); !errors.Is(err, domain.ErrUnsupportedSource) {
		t.Fatalf("ExportDocument error = %v, want ErrUnsupportedSource", err)
	}
	if len(sink.names) != 0 {
		t.Fatalf("a refused export wrote %v", sink.names)
	}
}
