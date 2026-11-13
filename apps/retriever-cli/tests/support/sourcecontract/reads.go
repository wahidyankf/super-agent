package sourcecontract

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// runReads asserts the catalog-read half of the contract: a gateway must read the seven operations of the catalog with
// the Records the rows declare. The expected values are literals, as in Run. A gateway that is not a SourceReader fails
// the first subtest.
func runReads(t *testing.T, newGateway func(t *testing.T, w Wiring) ports.SourceGateway) {
	t.Helper()
	standard := StandardDataset()
	reader := func(t *testing.T, w Wiring) ports.SourceReader {
		t.Helper()
		r, ok := newGateway(t, w).(ports.SourceReader)
		if !ok {
			t.Fatalf("gateway does not read catalog operations: it is not a ports.SourceReader")
		}
		return r
	}
	read := func(t *testing.T, r ports.SourceReader, op domain.OperationID, args map[string]any) domain.Record {
		t.Helper()
		rec, err := r.Read(context.Background(), op, domain.NewArgs(args))
		if err != nil {
			t.Fatalf("Read(%s): %v", op, err)
		}
		return rec
	}
	expectValue := func(t *testing.T, got domain.Record, want any) {
		t.Helper()
		if gotValue := asValue(got); !reflect.DeepEqual(gotValue, want) {
			t.Fatalf("Read = %#v, want %#v", gotValue, want)
		}
	}

	t.Run("reads every collection as a list of id and name", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		want := make([]any, 0, len(standard.Collections))
		for _, c := range standard.Collections {
			want = append(want, map[string]any{"id": c.ID, "name": c.Name})
		}
		expectValue(t, read(t, r, domain.OpCollectionsList, nil), want)
	})

	t.Run("reads one collection's id and name", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		expectValue(t, read(t, r, domain.OpCollectionsInfo, map[string]any{"collectionId": "col-tree"}),
			map[string]any{"id": "col-tree", "name": "Tree Collection"})
	})

	t.Run("reads an unknown collection as not found", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		_, err := r.Read(context.Background(), domain.OpCollectionsInfo, domain.NewArgs(map[string]any{"collectionId": "col-missing"}))
		if !errors.Is(err, domain.ErrCollectionNotFound) {
			t.Fatalf("Read(collections.info) error = %v, want ErrCollectionNotFound", err)
		}
	})

	t.Run("reads a collection tree as nodes with parent and depth", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		expectValue(t, read(t, r, domain.OpCollectionsDocuments, map[string]any{"collectionId": "col-tree"}), []any{
			map[string]any{"id": "doc-1", "title": "Getting Started", "url": "/doc/getting-started", "parentId": nil, "depth": 0},
			map[string]any{"id": "doc-2", "title": "Setup", "url": "/doc/setup", "parentId": "doc-1", "depth": 1},
			map[string]any{"id": "doc-3", "title": "Install", "url": "/doc/install", "parentId": "doc-2", "depth": 2},
			map[string]any{"id": "doc-4", "title": "Access", "url": "/doc/access", "parentId": "doc-1", "depth": 1},
			map[string]any{"id": "doc-5", "title": "Policies", "url": "/doc/policies", "parentId": nil, "depth": 0},
		})
	})

	t.Run("reads an empty collection tree as an empty list", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		expectValue(t, read(t, r, domain.OpCollectionsDocuments, map[string]any{"collectionId": "col-empty"}), []any{})
	})

	t.Run("reads a document with its markdown", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		expectValue(t, read(t, r, domain.OpDocumentsGet, map[string]any{"documentId": "doc-2"}), map[string]any{
			"id":           "doc-2",
			"title":        "Setup",
			"url":          "/doc/setup",
			"collectionId": "col-tree",
			"revision":     1,
			"markdown":     "# Setup\n\nRun the steps.\n",
		})
	})

	t.Run("reads an unknown document as not found", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		_, err := r.Read(context.Background(), domain.OpDocumentsGet, domain.NewArgs(map[string]any{"documentId": "doc-missing"}))
		if !errors.Is(err, domain.ErrDocumentNotFound) {
			t.Fatalf("Read(documents.get) error = %v, want ErrDocumentNotFound", err)
		}
	})

	t.Run("lists one page of documents in dataset order", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		expectValue(t, read(t, r, domain.OpDocumentsList, map[string]any{"offset": 1, "limit": 2}), []any{
			map[string]any{"id": "doc-2", "title": "Setup", "url": "/doc/setup", "collectionId": "col-tree"},
			map[string]any{"id": "doc-3", "title": "Install", "url": "/doc/install", "collectionId": "col-tree"},
		})
	})

	// The collection filter of documents.list takes a UUID (the binder refuses any other text), and the dataset's
	// readable IDs are not UUIDs. The filter is therefore covered by the CLI and MCP suites, which use UUIDs.

	t.Run("searches documents by title and text, without regard to case", func(t *testing.T) {
		r := reader(t, Wiring{Dataset: standard})
		expectValue(t, read(t, r, domain.OpDocumentsSearch, map[string]any{"query": "INSTALL", "offset": 0, "limit": 25}), []any{
			map[string]any{"id": "doc-3", "title": "Install", "url": "/doc/install", "collectionId": "col-tree"},
		})
	})

	t.Run("reports each fault on every read", func(t *testing.T) {
		faults := []struct {
			fault Fault
			match func(error) bool
			name  string
		}{
			{FaultKeyRejected, func(err error) bool { return errors.Is(err, domain.ErrKeyRejected) }, "ErrKeyRejected"},
			{FaultAccessDenied, func(err error) bool { return errors.Is(err, domain.ErrAccessDenied) }, "ErrAccessDenied"},
			{FaultUpstream, func(err error) bool { return errors.Is(err, domain.ErrUpstreamFailure) }, "ErrUpstreamFailure"},
		}
		for _, f := range faults {
			r := reader(t, Wiring{Dataset: standard, Fault: f.fault})
			_, err := r.Read(context.Background(), domain.OpCollectionsList, domain.NewArgs(nil))
			if !f.match(err) {
				t.Fatalf("%s: Read(collections.list) error = %v, want %s", f.name, err, f.name)
			}
		}
	})
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
