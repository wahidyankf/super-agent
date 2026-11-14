package sourcecontract

import (
	"context"
	"reflect"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// readChecks lists the catalog-read assertions: a gateway must read the seven operations of the catalog with the
// Records the rows declare. The expected values are literals, as in the checks of contract.go. A gateway that is not a
// SourceReader fails the first read check.
func (s suite) readChecks() []check {
	return []check{
		{"reads every collection as a list of id and name", s.readsCollections},
		{"reads one collection's id and name", s.readsOneCollection},
		{"reads an unknown collection as not found", s.readsUnknownCollection},
		{"reads a collection tree as nodes with parent and depth", s.readsTree},
		{"reads an empty collection tree as an empty list", s.readsEmptyTree},
		{"reads a document with its markdown", s.readsDocument},
		{"reads an unknown document as not found", s.readsUnknownDocument},
		{"lists one page of documents in dataset order", s.readsPage},
		// The collection filter of documents.list takes a UUID (the binder refuses any other text), and the dataset's
		// readable IDs are not UUIDs. The filter is therefore covered by the CLI and MCP suites, which use UUIDs.
		{"searches documents by title and text, without regard to case", s.readsSearch},
		{"reports each fault on every read", s.readsFaults},
	}
}

// readsCollections checks that collections.list returns every standard collection as an id and a name.
func (s suite) readsCollections(open opener, r reporter) {
	want := make([]any, 0, len(s.standard.Collections))
	for _, c := range s.standard.Collections {
		want = append(want, map[string]any{"id": c.ID, "name": c.Name})
	}
	expectRead(open, r, Wiring{Dataset: s.standard}, domain.OpCollectionsList, nil, want)
}

// readsOneCollection checks that collections.info returns the id and name of col-tree.
func (s suite) readsOneCollection(open opener, r reporter) {
	expectRead(open, r, Wiring{Dataset: s.standard}, domain.OpCollectionsInfo, map[string]any{"collectionId": "col-tree"},
		map[string]any{"id": "col-tree", "name": "Tree Collection"})
}

// readsUnknownCollection checks that collections.info of a missing collection fails as not found.
func (s suite) readsUnknownCollection(open opener, r reporter) {
	expectReadFailure(open, r, Wiring{Dataset: s.standard}, domain.OpCollectionsInfo, map[string]any{"collectionId": "col-missing"},
		isError(domain.ErrCollectionNotFound), "ErrCollectionNotFound")
}

// readsTree checks that collections.documents returns the tree of col-tree, depth first, with parent and depth.
func (s suite) readsTree(open opener, r reporter) {
	expectRead(open, r, Wiring{Dataset: s.standard}, domain.OpCollectionsDocuments, map[string]any{"collectionId": "col-tree"}, []any{
		map[string]any{"id": "doc-1", "title": "Getting Started", "url": "/doc/getting-started", "parentId": nil, "depth": 0},
		map[string]any{"id": "doc-2", "title": "Setup", "url": "/doc/setup", "parentId": "doc-1", "depth": 1},
		map[string]any{"id": "doc-3", "title": "Install", "url": "/doc/install", "parentId": "doc-2", "depth": 2},
		map[string]any{"id": "doc-4", "title": "Access", "url": "/doc/access", "parentId": "doc-1", "depth": 1},
		map[string]any{"id": "doc-5", "title": "Policies", "url": "/doc/policies", "parentId": nil, "depth": 0},
	})
}

// readsEmptyTree checks that collections.documents of col-empty returns an empty list.
func (s suite) readsEmptyTree(open opener, r reporter) {
	expectRead(open, r, Wiring{Dataset: s.standard}, domain.OpCollectionsDocuments, map[string]any{"collectionId": "col-empty"}, []any{})
}

// readsDocument checks that documents.get returns doc-2 with its markdown.
func (s suite) readsDocument(open opener, r reporter) {
	expectRead(open, r, Wiring{Dataset: s.standard}, domain.OpDocumentsGet, map[string]any{"documentId": "doc-2"}, map[string]any{
		"id":           "doc-2",
		"title":        "Setup",
		"url":          "/doc/setup",
		"collectionId": "col-tree",
		"revision":     1,
		"markdown":     "# Setup\n\nRun the steps.\n",
	})
}

// readsUnknownDocument checks that documents.get of a missing document fails as not found.
func (s suite) readsUnknownDocument(open opener, r reporter) {
	expectReadFailure(open, r, Wiring{Dataset: s.standard}, domain.OpDocumentsGet, map[string]any{"documentId": "doc-missing"},
		isError(domain.ErrDocumentNotFound), "ErrDocumentNotFound")
}

// readsPage checks that documents.list returns the second and third documents for offset 1 and limit 2.
func (s suite) readsPage(open opener, r reporter) {
	expectRead(open, r, Wiring{Dataset: s.standard}, domain.OpDocumentsList, map[string]any{"offset": 1, "limit": 2}, []any{
		map[string]any{"id": "doc-2", "title": "Setup", "url": "/doc/setup", "collectionId": "col-tree"},
		map[string]any{"id": "doc-3", "title": "Install", "url": "/doc/install", "collectionId": "col-tree"},
	})
}

// readsSearch checks that documents.search finds a title whatever the case of the query.
func (s suite) readsSearch(open opener, r reporter) {
	expectRead(open, r, Wiring{Dataset: s.standard}, domain.OpDocumentsSearch, map[string]any{"query": "INSTALL", "offset": 0, "limit": 25}, []any{
		map[string]any{"id": "doc-3", "title": "Install", "url": "/doc/install", "collectionId": "col-tree"},
	})
}

// readsFaults checks that a read fails with each fault's error when the source answers with that fault.
func (s suite) readsFaults(open opener, r reporter) {
	faults := []struct {
		fault Fault
		match func(error) bool
		name  string
	}{
		{FaultKeyRejected, isError(domain.ErrKeyRejected), "ErrKeyRejected"},
		{FaultAccessDenied, isError(domain.ErrAccessDenied), "ErrAccessDenied"},
		{FaultUpstream, isError(domain.ErrUpstreamFailure), "ErrUpstreamFailure"},
	}
	for _, f := range faults {
		expectReadFailure(open, r, Wiring{Dataset: s.standard, Fault: f.fault}, domain.OpCollectionsList, nil, f.match, f.name)
	}
}

// readerOf returns the reader that the gateway for the wiring is, or reports that the gateway reads no catalog operation.
func readerOf(open opener, r reporter, w Wiring) (ports.SourceReader, bool) {
	r.Helper()
	reader, ok := open(w).(ports.SourceReader)
	if !ok {
		r.Fatalf("gateway does not read catalog operations: it is not a ports.SourceReader")
		return nil, false
	}
	return reader, true
}

// expectRead runs one catalog operation and checks that it returns want, with each Record shown as plain Go values.
func expectRead(open opener, r reporter, w Wiring, op domain.OperationID, args map[string]any, want any) {
	r.Helper()
	reader, ok := readerOf(open, r, w)
	if !ok {
		return
	}
	rec, err := reader.Read(context.Background(), op, domain.NewArgs(args))
	if err != nil {
		r.Fatalf("Read(%s): %v", op, err)
		return
	}
	if got := asValue(rec); !reflect.DeepEqual(got, want) {
		r.Fatalf("Read = %#v, want %#v", got, want)
	}
}

// expectReadFailure runs one catalog operation and checks that its error is one that match accepts.
func expectReadFailure(open opener, r reporter, w Wiring, op domain.OperationID, args map[string]any, match func(error) bool, want string) {
	r.Helper()
	reader, ok := readerOf(open, r, w)
	if !ok {
		return
	}
	_, err := reader.Read(context.Background(), op, domain.NewArgs(args))
	if !match(err) {
		r.Fatalf("Read(%s) error = %v, want %s", op, err, want)
	}
}

// asValue returns a Record as plain Go values, so a test compares it with a literal: an object is a map, and a list is
// a slice of those maps. A nested Record is converted too.
func asValue(rec domain.Record) any {
	if rec.IsList() {
		out := make([]any, 0, len(rec.Items()))
		for _, item := range rec.Items() {
			out = append(out, asValue(item))
		}
		return out
	}
	out := map[string]any{}
	for _, f := range rec.Fields() {
		if nested, ok := f.Value.(domain.Record); ok {
			out[f.Name] = asValue(nested)
			continue
		}
		out[f.Name] = f.Value
	}
	return out
}
