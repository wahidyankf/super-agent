package outline

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
)

// sampleOptions is a small wiki: one collection with a parent and a child document.
func sampleOptions() []outlinestub.Option {
	return []outlinestub.Option{
		outlinestub.WithCollections(outlinestub.Collection{ID: "col-1", Name: "Sample Collection"}),
		outlinestub.WithDocuments(
			outlinestub.Document{
				ID: "doc-1", CollectionID: "col-1", Title: "Getting Started",
				URL: "/doc/getting-started", Markdown: "# Getting Started\n\nWelcome.\n",
			},
			outlinestub.Document{
				ID: "doc-2", CollectionID: "col-1", ParentID: "doc-1", Title: "Setup",
				URL: "/doc/setup", Markdown: "# Setup\n",
			},
		),
	}
}

func TestListCollectionsReadsPagesOfTwentyFiveUntilAShortPage(t *testing.T) {
	r := newRig(t, []outlinestub.Option{outlinestub.WithCollections(outlinestub.NumberedCollections(60)...)})

	got, err := r.gw.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(got) != 60 {
		t.Fatalf("ListCollections returned %d collections, want 60", len(got))
	}
	reqs := r.stub.Requests()
	if len(reqs) != 3 {
		t.Fatalf("requests = %d, want 3 (offsets 0, 25, 50)", len(reqs))
	}
	wantOffsets := []int{0, 25, 50}
	for i, req := range reqs {
		if req.Method != "collections.list" {
			t.Fatalf("request %d method = %q, want collections.list", i, req.Method)
		}
		if req.Limit != 25 {
			t.Fatalf("request %d limit = %d, want 25", i, req.Limit)
		}
		if req.Offset != wantOffsets[i] {
			t.Fatalf("request %d offset = %d, want %d", i, req.Offset, wantOffsets[i])
		}
	}
}

func TestListCollectionsStopsAtThePageCap(t *testing.T) {
	r := newRig(t, []outlinestub.Option{outlinestub.WithFullPages()})

	_, err := r.gw.ListCollections(context.Background())
	if !errors.Is(err, domain.ErrPageCapReached) {
		t.Fatalf("ListCollections error = %v, want ErrPageCapReached", err)
	}
	if n := r.stub.Calls("collections.list"); n != 20 {
		t.Fatalf("collections.list calls = %d, want 20 (the page cap)", n)
	}
}

func TestCollectionTreeAndDocumentRefuseZeroIDsBeforeAnyRequest(t *testing.T) {
	r := newRig(t, sampleOptions())
	ctx := context.Background()

	if _, err := r.gw.CollectionTree(ctx, domain.CollectionID{}); !errors.Is(err, domain.ErrInvalidCollectionID) {
		t.Fatalf("CollectionTree with a zero ID error = %v, want ErrInvalidCollectionID", err)
	}
	if _, err := r.gw.Document(ctx, domain.DocumentID{}); !errors.Is(err, domain.ErrInvalidDocumentID) {
		t.Fatalf("Document with a zero ID error = %v, want ErrInvalidDocumentID", err)
	}
	if n := len(r.stub.Requests()); n != 0 {
		t.Fatalf("requests = %d, want 0 for invalid IDs", n)
	}
}

// A version 2 reply carries the Markdown text, so the body is never read with documents.export.
func TestDocumentReadsTheTextOfInfoAndSkipsExport(t *testing.T) {
	r := newRig(t, sampleOptions())
	id := mustDocID(t, "doc-1")

	got, err := r.gw.Document(context.Background(), id)
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	want := domain.SourceDocument{
		ID:           "doc-1",
		Title:        "Getting Started",
		URL:          "/doc/getting-started",
		Markdown:     "# Getting Started\n\nWelcome.\n",
		CollectionID: "col-1",
	}
	if got != want {
		t.Fatalf("Document = %+v, want %+v", got, want)
	}
	if n := r.stub.Calls("documents.info"); n != 1 {
		t.Fatalf("documents.info calls = %d, want 1", n)
	}
	if n := r.stub.Calls("documents.export"); n != 0 {
		t.Fatalf("documents.export calls = %d, want 0", n)
	}
}

// editor data with no text sends the client to documents.export for the body.
func TestDocumentReadsTheBodyWithExportWhenInfoHasNoText(t *testing.T) {
	r := newRig(t, sampleOptions())
	r.stub.SetEditorOnlyDocuments(true)

	got, err := r.gw.Document(context.Background(), mustDocID(t, "doc-1"))
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if got.Markdown != "# Getting Started\n\nWelcome.\n" {
		t.Fatalf("Markdown = %q, want the exported text", got.Markdown)
	}
	if n := r.stub.Calls("documents.export"); n != 1 {
		t.Fatalf("documents.export calls = %d, want 1", n)
	}
}

func TestCollectionTreeSendsTheCollectionIDAsOneRequest(t *testing.T) {
	r := newRig(t, sampleOptions())

	nodes, err := r.gw.CollectionTree(context.Background(), mustColID(t, "col-1"))
	if err != nil {
		t.Fatalf("CollectionTree: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("CollectionTree returned %d nodes, want 2", len(nodes))
	}
	reqs := r.stub.Requests()
	if len(reqs) != 1 || reqs[0].Method != "collections.documents" || reqs[0].ID != "col-1" {
		t.Fatalf("requests = %+v, want one collections.documents call for col-1", reqs)
	}
}

// TestMalformedRepliesAreUnexpectedResponses covers every reply shape the client must
// refuse, one endpoint at a time. Each reply is refused with ErrUnexpectedResponse,
// and the refusal never repeats the body.
func TestMalformedRepliesAreUnexpectedResponses(t *testing.T) {
	cases := []struct {
		name   string
		method string
		body   string
		call   func(t *testing.T, r rig) error
	}{
		{
			name: "collections list without data", method: "collections.list",
			body: `{"pagination":{"offset":0,"limit":25}}`,
			call: func(t *testing.T, r rig) error { _, err := r.gw.ListCollections(context.Background()); return err },
		},
		{
			name: "collections list with null data", method: "collections.list",
			body: `{"data":null}`,
			call: func(t *testing.T, r rig) error { _, err := r.gw.ListCollections(context.Background()); return err },
		},
		{
			name: "collections list with a collection ID outside the allowed set", method: "collections.list",
			body: `{"data":[{"id":"bad.id","name":"x"}]}`,
			call: func(t *testing.T, r rig) error { _, err := r.gw.ListCollections(context.Background()); return err },
		},
		{
			name: "collections list that is not JSON", method: "collections.list",
			body: `<html>gateway page</html>`,
			call: func(t *testing.T, r rig) error { _, err := r.gw.ListCollections(context.Background()); return err },
		},
		{
			name: "tree without data", method: "collections.documents",
			body: `{}`,
			call: func(t *testing.T, r rig) error {
				_, err := r.gw.CollectionTree(context.Background(), mustColID(t, "col-1"))
				return err
			},
		},
		{
			name: "tree node with an invalid document ID", method: "collections.documents",
			body: `{"data":[{"id":"doc.1","title":"x","url":"/x","children":[]}]}`,
			call: func(t *testing.T, r rig) error {
				_, err := r.gw.CollectionTree(context.Background(), mustColID(t, "col-1"))
				return err
			},
		},
		{
			name: "tree child with an invalid document ID", method: "collections.documents",
			body: `{"data":[{"id":"doc-1","title":"x","url":"/x","children":[{"id":"doc;ls","title":"y","url":"/y","children":[]}]}]}`,
			call: func(t *testing.T, r rig) error {
				_, err := r.gw.CollectionTree(context.Background(), mustColID(t, "col-1"))
				return err
			},
		},
		{
			name: "info without data", method: "documents.info",
			body: `{"data":null}`,
			call: func(t *testing.T, r rig) error {
				_, err := r.gw.Document(context.Background(), mustDocID(t, "doc-1"))
				return err
			},
		},
		{
			name: "info with an invalid document ID", method: "documents.info",
			body: `{"data":{"id":"doc.1","title":"x","url":"/x","collectionId":"col-1","text":"t"}}`,
			call: func(t *testing.T, r rig) error {
				_, err := r.gw.Document(context.Background(), mustDocID(t, "doc-1"))
				return err
			},
		},
		{
			name: "export whose data is not a string", method: "documents.export",
			body: `{"data":42}`,
			call: func(t *testing.T, r rig) error {
				r.stub.SetEditorOnlyDocuments(true)
				_, err := r.gw.Document(context.Background(), mustDocID(t, "doc-1"))
				return err
			},
		},
		{
			name: "export without data", method: "documents.export",
			body: `{}`,
			call: func(t *testing.T, r rig) error {
				r.stub.SetEditorOnlyDocuments(true)
				_, err := r.gw.Document(context.Background(), mustDocID(t, "doc-1"))
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, sampleOptions())
			r.stub.SetFault(outlinestub.Fault{Method: tc.method, Status: 200, Body: tc.body})
			err := tc.call(t, r)
			if !errors.Is(err, domain.ErrUnexpectedResponse) {
				t.Fatalf("error = %v, want ErrUnexpectedResponse", err)
			}
		})
	}
}

// statusCase is one scripted failure and the sentinel each call must return for it.
type statusCase struct {
	name  string
	fault outlinestub.Fault
	list  error
	tree  error
	doc   error
}

// failureCases maps every failure the client must classify onto its sentinel.
func failureCases() []statusCase {
	return []statusCase{
		{name: "401 is a rejected key", fault: outlinestub.Fault{Status: 401},
			list: domain.ErrKeyRejected, tree: domain.ErrKeyRejected, doc: domain.ErrKeyRejected},
		{name: "403 is denied access", fault: outlinestub.Fault{Status: 403},
			list: domain.ErrAccessDenied, tree: domain.ErrAccessDenied, doc: domain.ErrAccessDenied},
		{name: "404 on the list is not found", fault: outlinestub.Fault{Status: 404},
			list: domain.ErrNotFound, tree: domain.ErrCollectionNotFound, doc: domain.ErrDocumentNotFound},
		{name: "500 is an upstream failure", fault: outlinestub.Fault{Status: 500},
			list: domain.ErrUpstreamFailure, tree: domain.ErrUpstreamFailure, doc: domain.ErrUpstreamFailure},
		{name: "418 is an unexpected response", fault: outlinestub.Fault{Status: 418},
			list: domain.ErrUnexpectedResponse, tree: domain.ErrUnexpectedResponse, doc: domain.ErrUnexpectedResponse},
		{name: "302 is a redirect", fault: outlinestub.Fault{Status: 302, Location: "https://evil.example.com/api/collections.list"},
			list: domain.ErrRedirected, tree: domain.ErrRedirected, doc: domain.ErrRedirected},
		{name: "a dropped connection is unreachable", fault: outlinestub.Fault{Drop: true},
			list: domain.ErrUnreachable, tree: domain.ErrUnreachable, doc: domain.ErrUnreachable},
		{name: "no reply in time is a timeout", fault: outlinestub.Fault{Hang: true},
			list: domain.ErrTimedOut, tree: domain.ErrTimedOut, doc: domain.ErrTimedOut},
	}
}

func TestEveryFailureMapsToItsSentinel(t *testing.T) {
	for _, tc := range failureCases() {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, sampleOptions())
			r.stub.SetFault(tc.fault)
			ctx := context.Background()

			if _, err := r.gw.ListCollections(ctx); !errors.Is(err, tc.list) {
				t.Fatalf("ListCollections error = %v, want %v", err, tc.list)
			}
			if _, err := r.gw.CollectionTree(ctx, mustColID(t, "col-1")); !errors.Is(err, tc.tree) {
				t.Fatalf("CollectionTree error = %v, want %v", err, tc.tree)
			}
			if _, err := r.gw.Document(ctx, mustDocID(t, "doc-1")); !errors.Is(err, tc.doc) {
				t.Fatalf("Document error = %v, want %v", err, tc.doc)
			}
		})
	}
}

func TestConcurrentCallsNeverOverlapOnTheWire(t *testing.T) {
	r := newRig(t, []outlinestub.Option{outlinestub.WithCollections(outlinestub.NumberedCollections(3)...)})

	const callers = 5
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.gw.ListCollections(context.Background())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent ListCollections: %v", err)
		}
	}
	if peak := r.stub.PeakInFlight(); peak != 1 {
		t.Fatalf("peak requests in flight = %d, want 1", peak)
	}
	if n := len(r.stub.Requests()); n != callers {
		t.Fatalf("requests = %d, want %d", n, callers)
	}
}

func mustColID(t *testing.T, raw string) domain.CollectionID {
	t.Helper()
	id, err := domain.NewCollectionID(raw)
	if err != nil {
		t.Fatalf("NewCollectionID(%q): %v", raw, err)
	}
	return id
}

func mustDocID(t *testing.T, raw string) domain.DocumentID {
	t.Helper()
	id, err := domain.NewDocumentID(raw)
	if err != nil {
		t.Fatalf("NewDocumentID(%q): %v", raw, err)
	}
	return id
}
