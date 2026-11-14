package sourcecontract

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// recorder is a reporter that keeps every mismatch, in order, instead of stopping the test that runs the check.
type recorder struct {
	msgs []string
}

func (r *recorder) Helper() {}

func (r *recorder) Fatalf(format string, args ...any) {
	r.msgs = append(r.msgs, fmt.Sprintf(format, args...))
}

// tampered is a gateway that answers from the fake for its Wiring. Each method passes the fake's answer through the
// function that a test sets for it, so one test gives the suite one defect. A nil function passes the answer on.
type tampered struct {
	Gateway
	list func([]domain.Collection, error) ([]domain.Collection, error)
	tree func([]domain.TreeNode, error) ([]domain.TreeNode, error)
	doc  func(domain.SourceDocument, error) (domain.SourceDocument, error)
	read func(domain.Record, error) (domain.Record, error)
}

func (m tampered) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	out, err := m.Gateway.ListCollections(ctx)
	if m.list == nil {
		return out, err
	}
	return m.list(out, err)
}

func (m tampered) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	out, err := m.Gateway.CollectionTree(ctx, id)
	if m.tree == nil {
		return out, err
	}
	return m.tree(out, err)
}

func (m tampered) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	out, err := m.Gateway.Document(ctx, id)
	if m.doc == nil {
		return out, err
	}
	return m.doc(out, err)
}

func (m tampered) Read(ctx context.Context, op domain.OperationID, a domain.Args) (domain.Record, error) {
	out, err := m.Gateway.Read(ctx, op, a)
	if m.read == nil {
		return out, err
	}
	return m.read(out, err)
}

// plain is a gateway that serves the three gateway calls and no catalog read.
type plain struct {
	ports.SourceGateway
}

// tampering returns an opener whose gateway is the fake for the Wiring, passed through the defects of m.
func tampering(m tampered) opener {
	return func(w Wiring) ports.SourceGateway {
		m.Gateway = NewFake(w)
		return m
	}
}

// fakeOpener returns an opener of the unchanged fake.
func fakeOpener() opener {
	return tampering(tampered{})
}

// Defects that replace a gateway's answer. The fail ones return a plain error, and the pass ones return no error.
var errBoom = errors.New("boom")

func failList([]domain.Collection, error) ([]domain.Collection, error) { return nil, errBoom }
func passList([]domain.Collection, error) ([]domain.Collection, error) { return nil, nil }
func failTree([]domain.TreeNode, error) ([]domain.TreeNode, error)     { return nil, errBoom }
func passTree([]domain.TreeNode, error) ([]domain.TreeNode, error)     { return nil, nil }
func failDoc(domain.SourceDocument, error) (domain.SourceDocument, error) {
	return domain.SourceDocument{}, errBoom
}
func passDoc(domain.SourceDocument, error) (domain.SourceDocument, error) {
	return domain.SourceDocument{}, nil
}
func failRead(domain.Record, error) (domain.Record, error) { return domain.Record{}, errBoom }
func passRead(domain.Record, error) (domain.Record, error) { return domain.Record{}, nil }

// profileRejecting returns the Outline profile, except that the source maps raw to an ID that the domain refuses.
func profileRejecting(raw string) Profile {
	p := OutlineProfile()
	p.SourceID = func(id string) string {
		if id == raw {
			return "not valid!"
		}
		return id
	}
	return p
}

// withoutDocument returns the standard dataset without the document that has the contract ID id.
func withoutDocument(id string) Dataset {
	d := StandardDataset()
	var kept []Document
	for _, doc := range d.Documents {
		if doc.ID != id {
			kept = append(kept, doc)
		}
	}
	d.Documents = kept
	return d
}

// mismatchCase is one defect that one check must report. A zero Profile or Dataset means the Outline profile and the
// standard dataset. want holds one fragment for each message the check reports, in order, and no other message.
type mismatchCase struct {
	name     string
	check    string
	p        Profile
	standard Dataset
	open     opener
	want     []string
}

func mismatchCases() []mismatchCase {
	return []mismatchCase{
		// lists every collection across pages
		{name: "list fails", check: "lists every collection across pages",
			open: tampering(tampered{list: failList}), want: []string{"ListCollections: boom"}},
		{name: "list is short", check: "lists every collection across pages",
			open: tampering(tampered{list: func(c []domain.Collection, err error) ([]domain.Collection, error) {
				return c[:1], err
			}}),
			want: []string{"ListCollections returned 1 collections, want 30"}},
		{name: "a name is wrong", check: "lists every collection across pages",
			open: tampering(tampered{list: func(c []domain.Collection, err error) ([]domain.Collection, error) {
				c[0].Name = "Wrong"
				return c, err
			}}),
			want: []string{`collection 0 = {"col-tree" "Wrong"}`}},

		// lists nothing for an empty wiki
		{name: "empty wiki list fails", check: "lists nothing for an empty wiki",
			open: tampering(tampered{list: failList}), want: []string{"ListCollections: boom"}},
		{name: "empty wiki lists a collection", check: "lists nothing for an empty wiki",
			open: tampering(tampered{list: func([]domain.Collection, error) ([]domain.Collection, error) {
				return []domain.Collection{{ID: "col-1", Name: "Stray"}}, nil
			}}),
			want: []string{"ListCollections returned 1 collections for an empty wiki, want 0"}},

		// flattens a tree depth first with parent and depth
		{name: "tree fails", check: "flattens a tree depth first with parent and depth",
			open: tampering(tampered{tree: failTree}), want: []string{"CollectionTree: boom"}},
		{name: "tree is short", check: "flattens a tree depth first with parent and depth",
			open: tampering(tampered{tree: func(n []domain.TreeNode, err error) ([]domain.TreeNode, error) {
				return n[:4], err
			}}),
			want: []string{"CollectionTree returned 4 nodes, want 5"}},
		{name: "a title is wrong", check: "flattens a tree depth first with parent and depth",
			open: tampering(tampered{tree: func(n []domain.TreeNode, err error) ([]domain.TreeNode, error) {
				n[0].Title = "Wrong"
				return n, err
			}}),
			want: []string{`node 0 = {"doc-1" "Wrong"`}},
		{name: "a root has a parent", check: "flattens a tree depth first with parent and depth",
			open: tampering(tampered{tree: func(n []domain.TreeNode, err error) ([]domain.TreeNode, error) {
				parent := "doc-5"
				n[0].ParentID = &parent
				return n, err
			}}),
			want: []string{`node 0 (doc-1) ParentID = "doc-5", want nil for a root`}},
		{name: "a child has the wrong parent", check: "flattens a tree depth first with parent and depth",
			open: tampering(tampered{tree: func(n []domain.TreeNode, err error) ([]domain.TreeNode, error) {
				parent := "doc-5"
				n[1].ParentID = &parent
				return n, err
			}}),
			want: []string{"node 1 (doc-2) ParentID ="}},
		{name: "the dataset lacks a tree document", check: "flattens a tree depth first with parent and depth",
			standard: withoutDocument("doc-1"),
			open: tampering(tampered{tree: func([]domain.TreeNode, error) ([]domain.TreeNode, error) {
				return make([]domain.TreeNode, 5), nil
			}}),
			want: []string{`the standard dataset has no document "doc-1"`}},
		{name: "tree collection ID is invalid", check: "flattens a tree depth first with parent and depth",
			p: profileRejecting("col-tree"), open: fakeOpener(),
			want: []string{`NewCollectionID("not valid!")`}},

		// returns an empty tree for a collection with no documents
		{name: "empty collection ID is invalid", check: "returns an empty tree for a collection with no documents",
			p: profileRejecting("col-empty"), open: fakeOpener(),
			want: []string{`NewCollectionID("not valid!")`}},
		{name: "empty tree fails", check: "returns an empty tree for a collection with no documents",
			open: tampering(tampered{tree: failTree}), want: []string{"CollectionTree: boom"}},
		{name: "empty tree has a node", check: "returns an empty tree for a collection with no documents",
			open: tampering(tampered{tree: func([]domain.TreeNode, error) ([]domain.TreeNode, error) {
				return []domain.TreeNode{{ID: "doc-1"}}, nil
			}}),
			want: []string{"CollectionTree returned 1 nodes for an empty collection, want 0"}},

		// reports an unknown collection as not found
		{name: "unknown collection ID is invalid", check: "reports an unknown collection as not found",
			p: profileRejecting("col-missing"), open: fakeOpener(),
			want: []string{`NewCollectionID("not valid!")`}},
		{name: "unknown collection answers no error", check: "reports an unknown collection as not found",
			open: tampering(tampered{tree: passTree}),
			want: []string{"CollectionTree error = <nil>, want ErrCollectionNotFound"}},

		// returns a document with its title, url, collection and markdown
		{name: "document ID is invalid", check: "returns a document with its title, url, collection and markdown",
			p: profileRejecting("doc-2"), open: fakeOpener(),
			want: []string{`NewDocumentID("not valid!")`}},
		{name: "the dataset lacks doc-2", check: "returns a document with its title, url, collection and markdown",
			standard: withoutDocument("doc-2"), open: fakeOpener(),
			want: []string{`the standard dataset has no document "doc-2"`}},
		{name: "document fails", check: "returns a document with its title, url, collection and markdown",
			open: tampering(tampered{doc: failDoc}), want: []string{"Document: boom"}},
		{name: "document has a wrong title", check: "returns a document with its title, url, collection and markdown",
			open: tampering(tampered{doc: func(d domain.SourceDocument, err error) (domain.SourceDocument, error) {
				d.Title = "Wrong"
				return d, err
			}}),
			want: []string{"Document = {ID:doc-2 Title:Wrong"}},

		// reports an unknown document as not found
		{name: "unknown document ID is invalid", check: "reports an unknown document as not found",
			p: profileRejecting("doc-missing"), open: fakeOpener(),
			want: []string{`NewDocumentID("not valid!")`}},
		{name: "unknown document answers no error", check: "reports an unknown document as not found",
			open: tampering(tampered{doc: passDoc}),
			want: []string{"Document error = <nil>, want ErrDocumentNotFound"}},

		// the fault checks share assertAllCalls
		{name: "rejected key: list answers no error", check: "reports a rejected key on every call",
			open: tampering(tampered{list: passList}),
			want: []string{"ListCollections error = <nil>, want ErrKeyRejected"}},
		{name: "rejected key: tree answers no error", check: "reports a rejected key on every call",
			open: tampering(tampered{tree: passTree}),
			want: []string{"CollectionTree error = <nil>, want ErrKeyRejected"}},
		{name: "rejected key: document answers no error", check: "reports a rejected key on every call",
			open: tampering(tampered{doc: passDoc}),
			want: []string{"Document error = <nil>, want ErrKeyRejected"}},
		{name: "rejected key: tree collection ID is invalid", check: "reports a rejected key on every call",
			p: profileRejecting("col-tree"), open: fakeOpener(),
			want: []string{`NewCollectionID("not valid!")`}},
		{name: "rejected key: document ID is invalid", check: "reports a rejected key on every call",
			p: profileRejecting("doc-1"), open: fakeOpener(),
			want: []string{`NewDocumentID("not valid!")`}},
		{name: "missing key names another variable", check: "reports a missing key by naming the variable on every call",
			open: tampering(tampered{list: func([]domain.Collection, error) ([]domain.Collection, error) {
				return nil, domain.MissingKeyError{Variable: "EXAMPLE_OTHER_API_KEY"}
			}}),
			want: []string{"want MissingKeyError naming EXAMPLE_OUTLINE_API_KEY"}},

		// the catalog reads
		{name: "gateway is not a reader", check: "reads every collection as a list of id and name",
			open: func(w Wiring) ports.SourceGateway { return plain{NewFake(w)} },
			want: []string{"gateway does not read catalog operations: it is not a ports.SourceReader"}},
		{name: "gateway is not a reader for a failure read", check: "reads an unknown collection as not found",
			open: func(w Wiring) ports.SourceGateway { return plain{NewFake(w)} },
			want: []string{"gateway does not read catalog operations: it is not a ports.SourceReader"}},
		{name: "read fails", check: "reads every collection as a list of id and name",
			open: tampering(tampered{read: failRead}), want: []string{"Read(collections.list): boom"}},
		{name: "read returns the wrong list", check: "reads every collection as a list of id and name",
			open: tampering(tampered{read: func(domain.Record, error) (domain.Record, error) {
				return domain.NewList(nil), nil
			}}),
			want: []string{"Read = []interface {}{}, want"}},
		{name: "read of a collection returns the wrong name", check: "reads one collection's id and name",
			open: tampering(tampered{read: func(domain.Record, error) (domain.Record, error) {
				return domain.NewRecord(domain.Field{Name: "id", Value: "col-tree"}, domain.Field{Name: "name", Value: "Wrong"}), nil
			}}),
			want: []string{`Read = map[string]interface {}{"id":"col-tree", "name":"Wrong"}, want`}},
		{name: "unknown collection read answers no error", check: "reads an unknown collection as not found",
			open: tampering(tampered{read: passRead}),
			want: []string{"Read(collections.info) error = <nil>, want ErrCollectionNotFound"}},
		{name: "unknown document read answers no error", check: "reads an unknown document as not found",
			open: tampering(tampered{read: passRead}),
			want: []string{"Read(documents.get) error = <nil>, want ErrDocumentNotFound"}},
		{name: "each read fault answers no error", check: "reports each fault on every read",
			open: tampering(tampered{read: passRead}),
			want: []string{
				"Read(collections.list) error = <nil>, want ErrKeyRejected",
				"Read(collections.list) error = <nil>, want ErrAccessDenied",
				"Read(collections.list) error = <nil>, want ErrUpstreamFailure",
			}},
	}
}

// findCheck returns the check of the suite with the given name.
func findCheck(t *testing.T, s suite, name string) check {
	t.Helper()
	for _, c := range s.checks() {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("the suite has no check named %q", name)
	return check{}
}

// TestChecksReportEachDefectOnce runs one check against one defect and asserts the exact messages it reports. A check
// stops at its first mismatch, so each defect yields the messages in want and nothing more.
func TestChecksReportEachDefectOnce(t *testing.T) {
	for _, tc := range mismatchCases() {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.p
			if p.SourceID == nil {
				p = OutlineProfile()
			}
			standard := tc.standard
			if standard.Collections == nil {
				standard = StandardDataset()
			}
			rec := &recorder{}
			findCheck(t, newSuite(p, standard), tc.check).run(tc.open, rec)
			if len(rec.msgs) != len(tc.want) {
				t.Fatalf("the check reported %d messages %q, want %d", len(rec.msgs), rec.msgs, len(tc.want))
			}
			for i, fragment := range tc.want {
				if !strings.Contains(rec.msgs[i], fragment) {
					t.Fatalf("message %d = %q, want it to contain %q", i, rec.msgs[i], fragment)
				}
			}
		})
	}
}

// TestEveryCheckPassesTheFake runs each check of the Outline suite against the unchanged fake. A passing check reports
// nothing, so this shows the checks accept the behaviour that the fake and the literals agree on.
func TestEveryCheckPassesTheFake(t *testing.T) {
	for _, c := range newSuite(OutlineProfile(), StandardDataset()).checks() {
		t.Run(c.name, func(t *testing.T) {
			rec := &recorder{}
			c.run(fakeOpener(), rec)
			if len(rec.msgs) != 0 {
				t.Fatalf("the fake fails the check: %q", rec.msgs)
			}
		})
	}
}

// TestAsValueShowsNestedRecordsAsPlainValues checks that a Record nested in a field, or in a list, is shown as plain
// Go values, so that a literal can be compared with it.
func TestAsValueShowsNestedRecordsAsPlainValues(t *testing.T) {
	rec := domain.NewRecord(
		domain.Field{Name: "outer", Value: domain.NewRecord(domain.Field{Name: "inner", Value: 7})},
		domain.Field{Name: "items", Value: domain.NewList([]domain.Record{domain.NewRecord(domain.Field{Name: "id", Value: "a"})})},
	)
	want := map[string]any{
		"outer": map[string]any{"inner": 7},
		"items": []any{map[string]any{"id": "a"}},
	}
	if got := asValue(rec); !reflect.DeepEqual(got, want) {
		t.Fatalf("asValue = %#v, want %#v", got, want)
	}
}
