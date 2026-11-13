// Package sourcecontract states the behaviour every ports.SourceGateway must show.
// One suite runs against each implementation: the in-memory fake in this package,
// the Outline adapter driven by its stub server, and the Kiwi adapter driven by its
// stub server. A gateway is built from a Wiring, which describes the dataset to serve
// and the failure to answer with.
//
// Expected values are written out as literals in this file. They do not come from
// any implementation, so an implementation cannot pass by sharing its own mistake.
// A source whose IDs, URLs, or Markdown differ from the standard dataset describes
// those differences in a Profile, and the suite maps its expectations through it.
package sourcecontract

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

// MissingKeyVariable is the variable a missing-key error names. The suite checks
// the error type, and a factory that builds a missing-key fault uses this name.
const MissingKeyVariable = "EXAMPLE_OUTLINE_API_KEY"

// Collection is one collection of a dataset.
type Collection struct {
	ID   string
	Name string
}

// Document is one document of a dataset. ParentID is "" for a root document.
// Siblings are listed in slice order, and that order is the tree order.
type Document struct {
	ID           string
	CollectionID string
	ParentID     string
	Title        string
	URL          string
	Markdown     string
}

// Dataset is the wiki a gateway must serve.
type Dataset struct {
	Collections []Collection
	Documents   []Document
}

// Fault names a failure the source must answer every call with.
type Fault int

const (
	// FaultNone means the source serves the dataset normally.
	FaultNone Fault = iota
	// FaultKeyRejected answers every call as HTTP 401, which maps to domain.ErrKeyRejected.
	FaultKeyRejected
	// FaultAccessDenied answers every call as HTTP 403, which maps to domain.ErrAccessDenied.
	FaultAccessDenied
	// FaultMissingKey gives no key, which maps to a domain.MissingKeyError.
	FaultMissingKey
	// FaultUpstream answers every call as HTTP 500, which maps to domain.ErrUpstreamFailure.
	FaultUpstream
)

// Wiring is what the suite gives a factory: the dataset to serve and the fault to answer with.
type Wiring struct {
	Dataset Dataset
	Fault   Fault
}

// Profile describes one source to the suite. The suite states its expectations with the
// standard dataset's IDs, and the profile maps them to the source's own.
type Profile struct {
	// MissingKeyVariable is the variable that a missing-key error must name.
	MissingKeyVariable string
	// SourceID maps a standard-dataset ID, such as col-tree or doc-2, to the source's ID for the same object. It also
	// maps the IDs that name a missing object, so those are absent from the source as well.
	SourceID func(contractID string) string
	// URL is the URL that the source reports for a document.
	URL func(d Document) string
	// Markdown is the Markdown that the source reports for a document.
	Markdown func(d Document) string
	// Catalog runs the read operations of the Outline catalog. Only the Outline source serves them.
	Catalog bool
}

// OutlineProfile is the profile of the Outline source, whose IDs, URLs, and Markdown are the standard dataset's own.
func OutlineProfile() Profile {
	return Profile{
		MissingKeyVariable: MissingKeyVariable,
		SourceID:           func(id string) string { return id },
		URL:                func(d Document) string { return d.URL },
		Markdown:           func(d Document) string { return d.Markdown },
		Catalog:            true,
	}
}

// StandardDataset is the wiki the main cases run against. It has 30 collections,
// so listing needs two pages of 25 and a short page. Collection col-tree holds a
// three-level tree, and collection col-empty holds no documents.
func StandardDataset() Dataset {
	collections := []Collection{
		{ID: "col-tree", Name: "Tree Collection"},
		{ID: "col-empty", Name: "Empty Collection"},
	}
	for i := 3; i <= 30; i++ {
		collections = append(collections, Collection{
			ID:   fmt.Sprintf("col-%02d", i),
			Name: fmt.Sprintf("Collection %02d", i),
		})
	}
	return Dataset{
		Collections: collections,
		Documents: []Document{
			{ID: "doc-1", CollectionID: "col-tree", Title: "Getting Started", URL: "/doc/getting-started", Markdown: "# Getting Started\n\nWelcome.\n"},
			{ID: "doc-2", CollectionID: "col-tree", ParentID: "doc-1", Title: "Setup", URL: "/doc/setup", Markdown: "# Setup\n\nRun the steps.\n"},
			{ID: "doc-3", CollectionID: "col-tree", ParentID: "doc-2", Title: "Install", URL: "/doc/install", Markdown: "# Install\n\nInstall it.\n"},
			{ID: "doc-4", CollectionID: "col-tree", ParentID: "doc-1", Title: "Access", URL: "/doc/access", Markdown: "# Access\n\nAsk for access.\n"},
			{ID: "doc-5", CollectionID: "col-tree", Title: "Policies", URL: "/doc/policies", Markdown: "# Policies\n\nRead the policies.\n"},
			{ID: "doc-6", CollectionID: "col-03", Title: "Elsewhere", URL: "/doc/elsewhere", Markdown: "# Elsewhere\n\nAnother collection.\n"},
		},
	}
}

// Run asserts the SourceGateway contract for the Outline source. newGateway builds a fresh gateway for
// each subtest from a Wiring, and registers any cleanup on the subtest's t.
func Run(t *testing.T, newGateway func(t *testing.T, w Wiring) ports.SourceGateway) {
	t.Helper()
	RunWith(t, OutlineProfile(), newGateway)
}

// RunWith asserts the SourceGateway contract for the source that p describes.
func RunWith(t *testing.T, p Profile, newGateway func(t *testing.T, w Wiring) ports.SourceGateway) {
	t.Helper()
	standard := StandardDataset()

	t.Run("lists every collection across pages", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard})
		got, err := gw.ListCollections(context.Background())
		if err != nil {
			t.Fatalf("ListCollections: %v", err)
		}
		if len(got) != len(standard.Collections) {
			t.Fatalf("ListCollections returned %d collections, want %d", len(got), len(standard.Collections))
		}
		for i, want := range standard.Collections {
			if got[i].ID != p.SourceID(want.ID) || got[i].Name != want.Name {
				t.Fatalf("collection %d = {%q %q}, want {%q %q}", i, got[i].ID, got[i].Name, p.SourceID(want.ID), want.Name)
			}
		}
	})

	t.Run("lists nothing for an empty wiki", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: Dataset{}})
		got, err := gw.ListCollections(context.Background())
		if err != nil {
			t.Fatalf("ListCollections: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("ListCollections returned %d collections for an empty wiki, want 0", len(got))
		}
	})

	t.Run("flattens a tree depth first with parent and depth", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard})
		got, err := gw.CollectionTree(context.Background(), mustCollectionID(t, p.SourceID("col-tree")))
		if err != nil {
			t.Fatalf("CollectionTree: %v", err)
		}
		want := []struct {
			id, title string
			parent    string // "" means a root, with a nil ParentID
			depth     int
		}{
			{"doc-1", "Getting Started", "", 0},
			{"doc-2", "Setup", "doc-1", 1},
			{"doc-3", "Install", "doc-2", 2},
			{"doc-4", "Access", "doc-1", 1},
			{"doc-5", "Policies", "", 0},
		}
		if len(got) != len(want) {
			t.Fatalf("CollectionTree returned %d nodes, want %d", len(got), len(want))
		}
		for i, w := range want {
			node := got[i]
			doc := mustDocument(t, standard, w.id)
			if node.ID != p.SourceID(w.id) || node.Title != w.title || node.URL != p.URL(doc) || node.Depth != w.depth {
				t.Fatalf("node %d = {%q %q %q depth %d}, want {%q %q %q depth %d}",
					i, node.ID, node.Title, node.URL, node.Depth, p.SourceID(w.id), w.title, p.URL(doc), w.depth)
			}
			if w.parent == "" {
				if node.ParentID != nil {
					t.Fatalf("node %d (%s) ParentID = %q, want nil for a root", i, w.id, *node.ParentID)
				}
				continue
			}
			if node.ParentID == nil || *node.ParentID != p.SourceID(w.parent) {
				t.Fatalf("node %d (%s) ParentID = %v, want %q", i, w.id, node.ParentID, p.SourceID(w.parent))
			}
		}
	})

	t.Run("returns an empty tree for a collection with no documents", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard})
		got, err := gw.CollectionTree(context.Background(), mustCollectionID(t, p.SourceID("col-empty")))
		if err != nil {
			t.Fatalf("CollectionTree: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("CollectionTree returned %d nodes for an empty collection, want 0", len(got))
		}
	})

	t.Run("reports an unknown collection as not found", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard})
		_, err := gw.CollectionTree(context.Background(), mustCollectionID(t, p.SourceID("col-missing")))
		if !errors.Is(err, domain.ErrCollectionNotFound) {
			t.Fatalf("CollectionTree error = %v, want ErrCollectionNotFound", err)
		}
	})

	t.Run("returns a document with its title, url, collection and markdown", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard})
		got, err := gw.Document(context.Background(), mustDocumentID(t, p.SourceID("doc-2")))
		if err != nil {
			t.Fatalf("Document: %v", err)
		}
		doc := mustDocument(t, standard, "doc-2")
		want := domain.SourceDocument{
			ID:           p.SourceID("doc-2"),
			Title:        "Setup",
			URL:          p.URL(doc),
			Markdown:     p.Markdown(doc),
			CollectionID: p.SourceID("col-tree"),
		}
		if got != want {
			t.Fatalf("Document = %+v, want %+v", got, want)
		}
	})

	t.Run("reports an unknown document as not found", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard})
		_, err := gw.Document(context.Background(), mustDocumentID(t, p.SourceID("doc-missing")))
		if !errors.Is(err, domain.ErrDocumentNotFound) {
			t.Fatalf("Document error = %v, want ErrDocumentNotFound", err)
		}
	})

	t.Run("reports a rejected key on every call", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard, Fault: FaultKeyRejected})
		assertAllCalls(t, p, gw, func(err error) bool { return errors.Is(err, domain.ErrKeyRejected) }, "ErrKeyRejected")
	})

	t.Run("reports denied access on every call", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard, Fault: FaultAccessDenied})
		assertAllCalls(t, p, gw, func(err error) bool { return errors.Is(err, domain.ErrAccessDenied) }, "ErrAccessDenied")
	})

	t.Run("reports a missing key by naming the variable on every call", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard, Fault: FaultMissingKey})
		assertAllCalls(t, p, gw, func(err error) bool {
			var missing domain.MissingKeyError
			return errors.As(err, &missing) && missing.Variable == p.MissingKeyVariable
		}, "MissingKeyError naming "+p.MissingKeyVariable)
	})

	t.Run("reports an upstream failure on every call", func(t *testing.T) {
		gw := newGateway(t, Wiring{Dataset: standard, Fault: FaultUpstream})
		assertAllCalls(t, p, gw, func(err error) bool { return errors.Is(err, domain.ErrUpstreamFailure) }, "ErrUpstreamFailure")
	})

	if p.Catalog {
		runReads(t, newGateway)
	}
}

// assertAllCalls checks that each of the three calls fails with an error that matches.
func assertAllCalls(t *testing.T, p Profile, gw ports.SourceGateway, match func(error) bool, want string) {
	t.Helper()
	ctx := context.Background()
	if _, err := gw.ListCollections(ctx); !match(err) {
		t.Fatalf("ListCollections error = %v, want %s", err, want)
	}
	if _, err := gw.CollectionTree(ctx, mustCollectionID(t, p.SourceID("col-tree"))); !match(err) {
		t.Fatalf("CollectionTree error = %v, want %s", err, want)
	}
	if _, err := gw.Document(ctx, mustDocumentID(t, p.SourceID("doc-1"))); !match(err) {
		t.Fatalf("Document error = %v, want %s", err, want)
	}
}

// mustDocument returns the standard dataset's document with the given contract ID.
func mustDocument(t *testing.T, d Dataset, id string) Document {
	t.Helper()
	for _, doc := range d.Documents {
		if doc.ID == id {
			return doc
		}
	}
	t.Fatalf("the standard dataset has no document %q", id)
	return Document{}
}

func mustCollectionID(t *testing.T, raw string) domain.CollectionID {
	t.Helper()
	id, err := domain.NewCollectionID(raw)
	if err != nil {
		t.Fatalf("NewCollectionID(%q): %v", raw, err)
	}
	return id
}

func mustDocumentID(t *testing.T, raw string) domain.DocumentID {
	t.Helper()
	id, err := domain.NewDocumentID(raw)
	if err != nil {
		t.Fatalf("NewDocumentID(%q): %v", raw, err)
	}
	return id
}
