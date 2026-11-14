package sourcecontract

import (
	"context"
	"strings"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
)

// Gateway is a source that serves both the gateway methods and the catalog reads. The Outline adapter and the
// in-memory fake are both Gateways.
type Gateway interface {
	ports.SourceGateway
	ports.SourceReader
}

// fake is an in-memory Gateway over a Wiring's dataset.
type fake struct {
	wiring Wiring
}

var _ Gateway = fake{}

// NewFake returns an in-memory gateway that serves the Wiring's dataset and answers
// with its fault. It is the unit-level implementation the contract suite runs against.
func NewFake(w Wiring) Gateway {
	return fake{wiring: w}
}

// failure returns the error the fault answers every call with, or nil.
func (f fake) failure() error {
	switch f.wiring.Fault {
	case FaultKeyRejected:
		return domain.ErrKeyRejected
	case FaultAccessDenied:
		return domain.ErrAccessDenied
	case FaultMissingKey:
		return domain.MissingKeyError{Variable: MissingKeyVariable}
	case FaultUpstream:
		return domain.ErrUpstreamFailure
	}
	return nil
}

// ListCollections returns the dataset's collections in order.
func (f fake) ListCollections(context.Context) ([]domain.Collection, error) {
	if err := f.failure(); err != nil {
		return nil, err
	}
	out := make([]domain.Collection, 0, len(f.wiring.Dataset.Collections))
	for _, c := range f.wiring.Dataset.Collections {
		out = append(out, domain.Collection{ID: c.ID, Name: c.Name})
	}
	return out, nil
}

// CollectionTree returns the documents of one collection, flattened depth first.
func (f fake) CollectionTree(_ context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	if err := f.failure(); err != nil {
		return nil, err
	}
	if !f.hasCollection(id.String()) {
		return nil, domain.ErrCollectionNotFound
	}
	return flattenTree(f.wiring.Dataset.Documents, id.String()), nil
}

// Document returns one document of the dataset.
func (f fake) Document(_ context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	if err := f.failure(); err != nil {
		return domain.SourceDocument{}, err
	}
	for _, d := range f.wiring.Dataset.Documents {
		if d.ID == id.String() {
			return domain.SourceDocument{
				ID:           d.ID,
				Title:        d.Title,
				URL:          d.URL,
				Markdown:     d.Markdown,
				CollectionID: d.CollectionID,
			}, nil
		}
	}
	return domain.SourceDocument{}, domain.ErrDocumentNotFound
}

// Read runs one catalog operation over the dataset. The arguments are the ones the catalog's binder checked, so a
// missing argument reads as its zero value. The whoami operation is not served, because the dataset holds no account.
func (f fake) Read(_ context.Context, op domain.OperationID, a domain.Args) (domain.Record, error) {
	if err := f.failure(); err != nil {
		return domain.Record{}, err
	}
	switch op {
	case domain.OpCollectionsList:
		items := make([]domain.Record, 0, len(f.wiring.Dataset.Collections))
		for _, c := range f.wiring.Dataset.Collections {
			items = append(items, collectionRecord(c.ID, c.Name))
		}
		return domain.NewList(items), nil
	case domain.OpCollectionsInfo:
		for _, c := range f.wiring.Dataset.Collections {
			if c.ID == a.String("collectionId") {
				return collectionRecord(c.ID, c.Name), nil
			}
		}
		return domain.Record{}, domain.ErrCollectionNotFound
	case domain.OpCollectionsDocuments:
		id := a.String("collectionId")
		if !f.hasCollection(id) {
			return domain.Record{}, domain.ErrCollectionNotFound
		}
		nodes := flattenTree(f.wiring.Dataset.Documents, id)
		items := make([]domain.Record, 0, len(nodes))
		for _, n := range nodes {
			var parent any
			if n.ParentID != nil {
				parent = *n.ParentID
			}
			items = append(items, domain.NewRecord(
				domain.Field{Name: "id", Value: n.ID},
				domain.Field{Name: "title", Value: n.Title},
				domain.Field{Name: "url", Value: n.URL},
				domain.Field{Name: "parentId", Value: parent},
				domain.Field{Name: "depth", Value: n.Depth},
			))
		}
		return domain.NewList(items), nil
	case domain.OpDocumentsGet:
		for _, d := range f.wiring.Dataset.Documents {
			if d.ID == a.String("documentId") {
				return domain.NewRecord(
					domain.Field{Name: "id", Value: d.ID},
					domain.Field{Name: "title", Value: d.Title},
					domain.Field{Name: "url", Value: d.URL},
					domain.Field{Name: "collectionId", Value: d.CollectionID},
					domain.Field{Name: "revision", Value: 1},
					domain.Field{Name: "markdown", Value: d.Markdown},
				), nil
			}
		}
		return domain.Record{}, domain.ErrDocumentNotFound
	case domain.OpDocumentsList:
		return f.documentPage(a.String("collectionId"), false, "", a.Int("offset"), a.Int("limit")), nil
	case domain.OpDocumentsSearch:
		return f.documentPage("", true, a.String("query"), a.Int("offset"), a.Int("limit")), nil
	}
	return domain.Record{}, domain.ErrUnsupportedOperation
}

// documentPage returns one page of the documents, in dataset order. A collection filter applies when collectionID is
// set. A search keeps the documents whose title or Markdown holds the query, without regard to case, as the Outline
// search does.
func (f fake) documentPage(collectionID string, search bool, query string, offset, limit int) domain.Record {
	var matches []Document
	for _, d := range f.wiring.Dataset.Documents {
		if collectionID != "" && d.CollectionID != collectionID {
			continue
		}
		if search && !containsFold(d.Title, query) && !containsFold(d.Markdown, query) {
			continue
		}
		matches = append(matches, d)
	}
	start := min(offset, len(matches))
	end := min(start+limit, len(matches))
	items := make([]domain.Record, 0, end-start)
	for _, d := range matches[start:end] {
		items = append(items, domain.NewRecord(
			domain.Field{Name: "id", Value: d.ID},
			domain.Field{Name: "title", Value: d.Title},
			domain.Field{Name: "url", Value: d.URL},
			domain.Field{Name: "collectionId", Value: d.CollectionID},
		))
	}
	return domain.NewList(items)
}

// collectionRecord returns the record of one collection.
func collectionRecord(id, name string) domain.Record {
	return domain.NewRecord(domain.Field{Name: "id", Value: id}, domain.Field{Name: "name", Value: name})
}

// containsFold reports whether s holds sub, without regard to case.
func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func (f fake) hasCollection(id string) bool {
	for _, c := range f.wiring.Dataset.Collections {
		if c.ID == id {
			return true
		}
	}
	return false
}

// flattenTree lists the documents of one collection depth first. Siblings keep
// their slice order, and a root has a nil ParentID.
func flattenTree(documents []Document, collectionID string) []domain.TreeNode {
	var out []domain.TreeNode
	var walk func(parentID string, depth int)
	walk = func(parentID string, depth int) {
		for _, d := range documents {
			if d.CollectionID != collectionID || d.ParentID != parentID {
				continue
			}
			node := domain.TreeNode{ID: d.ID, Title: d.Title, URL: d.URL, Depth: depth}
			if parentID != "" {
				parent := parentID
				node.ParentID = &parent
			}
			out = append(out, node)
			walk(d.ID, depth+1)
		}
	}
	walk("", 0)
	return out
}
