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

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
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

// RunWith asserts the SourceGateway contract for the source that p describes. Each assertion is a subtest, and
// newGateway builds the gateway for each subtest from that subtest's t.
func RunWith(t *testing.T, p Profile, newGateway func(t *testing.T, w Wiring) ports.SourceGateway) {
	t.Helper()
	for _, c := range newSuite(p, StandardDataset()).checks() {
		t.Run(c.name, func(t *testing.T) {
			t.Helper()
			c.run(func(w Wiring) ports.SourceGateway { return newGateway(t, w) }, t)
		})
	}
}

// reporter receives the first mismatch of a check. *testing.T satisfies it, so a subtest fails on the mismatch and
// stops there. A unit test passes a recorder instead, so each mismatch path runs without failing the test that checks it.
type reporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

// opener builds a fresh gateway for a Wiring.
type opener func(w Wiring) ports.SourceGateway

// check is one named assertion of the contract. It reports its first mismatch to the reporter and returns.
type check struct {
	name string
	run  func(open opener, r reporter)
}

// suite is what the checks need: the profile of the source, and the standard dataset that supplies the expectations.
type suite struct {
	p        Profile
	standard Dataset
}

// newSuite returns the checks for the source that p describes, with the expectations taken from standard.
func newSuite(p Profile, standard Dataset) suite {
	return suite{p: p, standard: standard}
}

// checks lists the assertions of the contract in order. The catalog reads follow when the source serves them.
func (s suite) checks() []check {
	list := []check{
		{"lists every collection across pages", s.listsEveryCollection},
		{"lists nothing for an empty wiki", s.listsNothingForEmptyWiki},
		{"flattens a tree depth first with parent and depth", s.flattensTree},
		{"returns an empty tree for a collection with no documents", s.emptyTree},
		{"reports an unknown collection as not found", s.unknownCollection},
		{"returns a document with its title, url, collection and markdown", s.returnsDocument},
		{"reports an unknown document as not found", s.unknownDocument},
		{"reports a rejected key on every call", s.keyRejected},
		{"reports denied access on every call", s.accessDenied},
		{"reports a missing key by naming the variable on every call", s.missingKey},
		{"reports an upstream failure on every call", s.upstreamFailure},
	}
	if s.p.Catalog {
		list = append(list, s.readChecks()...)
	}
	return list
}

// listsEveryCollection checks that ListCollections returns every standard collection, in order, under the source's IDs.
func (s suite) listsEveryCollection(open opener, r reporter) {
	got, err := open(Wiring{Dataset: s.standard}).ListCollections(context.Background())
	if err != nil {
		r.Fatalf("ListCollections: %v", err)
		return
	}
	if len(got) != len(s.standard.Collections) {
		r.Fatalf("ListCollections returned %d collections, want %d", len(got), len(s.standard.Collections))
		return
	}
	for i, want := range s.standard.Collections {
		if got[i].ID != s.p.SourceID(want.ID) || got[i].Name != want.Name {
			r.Fatalf("collection %d = {%q %q}, want {%q %q}", i, got[i].ID, got[i].Name, s.p.SourceID(want.ID), want.Name)
			return
		}
	}
}

// listsNothingForEmptyWiki checks that ListCollections over an empty dataset returns no collection and no error.
func (s suite) listsNothingForEmptyWiki(open opener, r reporter) {
	got, err := open(Wiring{Dataset: Dataset{}}).ListCollections(context.Background())
	if err != nil {
		r.Fatalf("ListCollections: %v", err)
		return
	}
	if len(got) != 0 {
		r.Fatalf("ListCollections returned %d collections for an empty wiki, want 0", len(got))
	}
}

// flattensTree checks that CollectionTree of col-tree lists its documents depth first, with parent IDs and depths.
func (s suite) flattensTree(open opener, r reporter) {
	id, ok := s.collectionID(r, s.p.SourceID("col-tree"))
	if !ok {
		return
	}
	got, err := open(Wiring{Dataset: s.standard}).CollectionTree(context.Background(), id)
	if err != nil {
		r.Fatalf("CollectionTree: %v", err)
		return
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
		r.Fatalf("CollectionTree returned %d nodes, want %d", len(got), len(want))
		return
	}
	for i, w := range want {
		doc, ok := s.document(r, w.id)
		if !ok {
			return
		}
		node := got[i]
		if node.ID != s.p.SourceID(w.id) || node.Title != w.title || node.URL != s.p.URL(doc) || node.Depth != w.depth {
			r.Fatalf("node %d = {%q %q %q depth %d}, want {%q %q %q depth %d}",
				i, node.ID, node.Title, node.URL, node.Depth, s.p.SourceID(w.id), w.title, s.p.URL(doc), w.depth)
			return
		}
		if w.parent == "" {
			if node.ParentID != nil {
				r.Fatalf("node %d (%s) ParentID = %q, want nil for a root", i, w.id, *node.ParentID)
				return
			}
			continue
		}
		if node.ParentID == nil || *node.ParentID != s.p.SourceID(w.parent) {
			r.Fatalf("node %d (%s) ParentID = %v, want %q", i, w.id, node.ParentID, s.p.SourceID(w.parent))
			return
		}
	}
}

// emptyTree checks that CollectionTree of col-empty returns no node.
func (s suite) emptyTree(open opener, r reporter) {
	id, ok := s.collectionID(r, s.p.SourceID("col-empty"))
	if !ok {
		return
	}
	got, err := open(Wiring{Dataset: s.standard}).CollectionTree(context.Background(), id)
	if err != nil {
		r.Fatalf("CollectionTree: %v", err)
		return
	}
	if len(got) != 0 {
		r.Fatalf("CollectionTree returned %d nodes for an empty collection, want 0", len(got))
	}
}

// unknownCollection checks that CollectionTree of a missing collection fails as not found.
func (s suite) unknownCollection(open opener, r reporter) {
	id, ok := s.collectionID(r, s.p.SourceID("col-missing"))
	if !ok {
		return
	}
	_, err := open(Wiring{Dataset: s.standard}).CollectionTree(context.Background(), id)
	if !errors.Is(err, domain.ErrCollectionNotFound) {
		r.Fatalf("CollectionTree error = %v, want ErrCollectionNotFound", err)
	}
}

// returnsDocument checks that Document of doc-2 returns its title, URL, collection, and Markdown.
func (s suite) returnsDocument(open opener, r reporter) {
	id, ok := s.documentID(r, s.p.SourceID("doc-2"))
	if !ok {
		return
	}
	doc, ok := s.document(r, "doc-2")
	if !ok {
		return
	}
	got, err := open(Wiring{Dataset: s.standard}).Document(context.Background(), id)
	if err != nil {
		r.Fatalf("Document: %v", err)
		return
	}
	want := domain.SourceDocument{
		ID:           s.p.SourceID("doc-2"),
		Title:        "Setup",
		URL:          s.p.URL(doc),
		Markdown:     s.p.Markdown(doc),
		CollectionID: s.p.SourceID("col-tree"),
	}
	if got != want {
		r.Fatalf("Document = %+v, want %+v", got, want)
	}
}

// unknownDocument checks that Document of a missing document fails as not found.
func (s suite) unknownDocument(open opener, r reporter) {
	id, ok := s.documentID(r, s.p.SourceID("doc-missing"))
	if !ok {
		return
	}
	_, err := open(Wiring{Dataset: s.standard}).Document(context.Background(), id)
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		r.Fatalf("Document error = %v, want ErrDocumentNotFound", err)
	}
}

// keyRejected checks that every call fails with ErrKeyRejected when the key is refused.
func (s suite) keyRejected(open opener, r reporter) {
	s.assertAllCalls(r, open(Wiring{Dataset: s.standard, Fault: FaultKeyRejected}), isError(domain.ErrKeyRejected), "ErrKeyRejected")
}

// accessDenied checks that every call fails with ErrAccessDenied when access is refused.
func (s suite) accessDenied(open opener, r reporter) {
	s.assertAllCalls(r, open(Wiring{Dataset: s.standard, Fault: FaultAccessDenied}), isError(domain.ErrAccessDenied), "ErrAccessDenied")
}

// missingKey checks that every call fails with a MissingKeyError that names the profile's variable.
func (s suite) missingKey(open opener, r reporter) {
	match := func(err error) bool {
		var missing domain.MissingKeyError
		return errors.As(err, &missing) && missing.Variable == s.p.MissingKeyVariable
	}
	s.assertAllCalls(r, open(Wiring{Dataset: s.standard, Fault: FaultMissingKey}), match, "MissingKeyError naming "+s.p.MissingKeyVariable)
}

// upstreamFailure checks that every call fails with ErrUpstreamFailure when the source answers with a server error.
func (s suite) upstreamFailure(open opener, r reporter) {
	s.assertAllCalls(r, open(Wiring{Dataset: s.standard, Fault: FaultUpstream}), isError(domain.ErrUpstreamFailure), "ErrUpstreamFailure")
}

// assertAllCalls checks that each of the three gateway calls fails with an error that match accepts.
func (s suite) assertAllCalls(r reporter, gw ports.SourceGateway, match func(error) bool, want string) {
	r.Helper()
	ctx := context.Background()
	if _, err := gw.ListCollections(ctx); !match(err) {
		r.Fatalf("ListCollections error = %v, want %s", err, want)
		return
	}
	id, ok := s.collectionID(r, s.p.SourceID("col-tree"))
	if !ok {
		return
	}
	if _, err := gw.CollectionTree(ctx, id); !match(err) {
		r.Fatalf("CollectionTree error = %v, want %s", err, want)
		return
	}
	docID, ok := s.documentID(r, s.p.SourceID("doc-1"))
	if !ok {
		return
	}
	if _, err := gw.Document(ctx, docID); !match(err) {
		r.Fatalf("Document error = %v, want %s", err, want)
	}
}

// isError returns a matcher that accepts an error which wraps target.
func isError(target error) func(error) bool {
	return func(err error) bool { return errors.Is(err, target) }
}

// collectionID returns the collection ID that raw names, or reports that the profile maps to an invalid one.
func (s suite) collectionID(r reporter, raw string) (domain.CollectionID, bool) {
	r.Helper()
	id, err := domain.NewCollectionID(raw)
	if err != nil {
		r.Fatalf("NewCollectionID(%q): %v", raw, err)
		return domain.CollectionID{}, false
	}
	return id, true
}

// documentID returns the document ID that raw names, or reports that the profile maps to an invalid one.
func (s suite) documentID(r reporter, raw string) (domain.DocumentID, bool) {
	r.Helper()
	id, err := domain.NewDocumentID(raw)
	if err != nil {
		r.Fatalf("NewDocumentID(%q): %v", raw, err)
		return domain.DocumentID{}, false
	}
	return id, true
}

// document returns the standard dataset's document with the contract ID, or reports that the dataset has none.
func (s suite) document(r reporter, id string) (Document, bool) {
	r.Helper()
	for _, doc := range s.standard.Documents {
		if doc.ID == id {
			return doc, true
		}
	}
	r.Fatalf("the standard dataset has no document %q", id)
	return Document{}, false
}
