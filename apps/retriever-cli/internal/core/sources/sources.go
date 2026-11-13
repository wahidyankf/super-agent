// Package sources holds the use cases that read external source wikis and export
// their documents through an export sink. Each use case validates its input before any
// gateway call, selects one configured source by the default rule, and reaches the
// wiki only through the ports.
package sources

import (
	"context"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources/operations"
)

// Service holds the two collaborators that every source use case needs. It never holds a gateway.
type Service struct {
	Catalog  domain.Catalog
	Gateways ports.GatewayResolver
}

// Capped returns the service with its catalog capped to ceiling. The gateway resolver is shared:
// a capped source is a different source value, and the resolver keeps its gateway apart by access level, so the
// gateway's own method allowlist follows the cap too.
func (s Service) Capped(ceiling domain.Access) Service {
	s.Catalog = s.Catalog.Capped(ceiling)
	return s
}

// ListCollections returns the collections of the selected source of the kind. A nil source applies the default rule.
func ListCollections(ctx context.Context, svc Service, kind domain.SourceKind, source *string) ([]domain.Collection, error) {
	src, err := selectSource(svc, kind, source)
	if err != nil {
		return nil, err
	}
	gw, err := svc.Gateways.For(src)
	if err != nil {
		return nil, err
	}
	return gw.ListCollections(ctx)
}

// ListCollectionTree returns the flat document tree of one collection of the selected source of the kind.
func ListCollectionTree(ctx context.Context, svc Service, kind domain.SourceKind, source *string, collectionID string) ([]domain.TreeNode, error) {
	src, err := selectSource(svc, kind, source)
	if err != nil {
		return nil, err
	}
	id, err := domain.NewCollectionID(collectionID)
	if err != nil {
		return nil, err
	}
	gw, err := svc.Gateways.For(src)
	if err != nil {
		return nil, err
	}
	return gw.CollectionTree(ctx, id)
}

// GetDocument returns one document of the selected source of the kind, with its Markdown body.
func GetDocument(ctx context.Context, svc Service, kind domain.SourceKind, source *string, documentID string) (domain.SourceDocument, error) {
	src, err := selectSource(svc, kind, source)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	id, err := domain.NewDocumentID(documentID)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	gw, err := svc.Gateways.For(src)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	return gw.Document(ctx, id)
}

// rowOf returns the catalog row of op and its source kind. An operation that the catalog does not hold, one with no
// known kind, or one that its kind does not support is ErrUnsupportedOperation, and it is refused before any source is
// selected. The kind of a row comes from its command path, so Read, Write, and Administer need no
// kind argument.
func rowOf(op domain.OperationID) (operations.Row, domain.SourceKind, error) {
	row, found := operations.Lookup(op)
	kind, known := row.Kind()
	if !found || !known || !operations.Supports(kind, row) {
		return operations.Row{}, domain.SourceKind{}, domain.ErrUnsupportedOperation
	}
	return row, kind, nil
}

// selectSource validates the optional source ID and applies the default rule for the kind: with an ID, the source must
// exist and have the kind; without one, exactly one source of the kind must exist. A nil source means the default rule.
func selectSource(svc Service, kind domain.SourceKind, source *string) (domain.Source, error) {
	var id *domain.SourceID
	if source != nil {
		parsed, err := domain.NewSourceID(*source)
		if err != nil {
			return domain.Source{}, err
		}
		id = &parsed
	}
	return svc.Catalog.Select(kind, id)
}

// Resolve returns the source of the kind that the optional source ID selects, by the default rule. It makes no gateway
// call, so an adapter can name the resolved source in its output before it reads the source.
func Resolve(svc Service, kind domain.SourceKind, source *string) (domain.Source, error) {
	return selectSource(svc, kind, source)
}
