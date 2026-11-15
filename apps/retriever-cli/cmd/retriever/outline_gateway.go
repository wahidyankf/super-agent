package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/outlineseam"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// collectionNotFoundReads are the engine-served reads whose 404 names a missing collection, and documentNotFoundReads
// are the ones whose 404 names a missing document, as the retired Go gateway mapped them. The engine reports both as
// a plain not found, so the gateway restores the sentinel the scenarios and the callers read. Any other engine
// failure passes through.
var (
	collectionNotFoundReads = map[domain.OperationID]bool{
		domain.OpCollectionsInfo:         true,
		domain.OpCollectionsDocuments:    true,
		domain.OpCollectionsMemberships:  true,
		domain.OpCollectionsGroupMembers: true,
	}
	documentNotFoundReads = map[domain.OperationID]bool{
		domain.OpDocumentsGet:              true,
		domain.OpDocumentsChildren:         true,
		domain.OpDocumentsInsights:         true,
		domain.OpDocumentsMemberships:      true,
		domain.OpDocumentsGroupMemberships: true,
		domain.OpDocumentsUsers:            true,
	}
)

// outlineGateway is the Outline gateway: the engine of the bundled outline.yml, with the not-found translation the
// retired Go gateway mapped. Every Outline operation runs on the engine: reads and writes through the spec, the
// legacy collection, tree, and document reads through the engine operations they name, and ListCollections through
// the engine role.
type outlineGateway struct {
	engine *engine.Engine
}

var (
	_ ports.SourceGateway = (*outlineGateway)(nil)
	_ ports.SourceReader  = (*outlineGateway)(nil)
	_ ports.SourceWriter  = (*outlineGateway)(nil)
)

// buildOutlineGateway returns the gateway of one Outline source: the engine of the bundled outline.yml. A source
// whose engine cannot be built stops the source.
func buildOutlineGateway(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	eseam engineSeam) (ports.SourceGateway, error) {
	eng, err := buildOutlineEngine(model, src, creds, seam, eseam)
	if err != nil {
		return nil, err
	}
	return &outlineGateway{engine: eng}, nil
}

// buildOutlineEngine returns the engine of the bundled outline.yml for one source. The seam, when it is on, replaces
// only the connection base: the transport rewrites each request to the seam base and round-trips with the seam
// client, so the engine reaches the stub. The seam's timeout bounds one engine request, so a run that gets no reply
// stops at the bound. Without a seam the engine is built as any other spec source is.
func buildOutlineEngine(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	eseam engineSeam) (*engine.Engine, error) {
	if !seam.on || seam.client == nil {
		return buildSpec(model, src, eseam, creds)
	}
	base, err := url.Parse(seam.base)
	if err != nil {
		return nil, err
	}
	if err := outlineseam.CheckBaseURL(seam.base); err != nil {
		return nil, err
	}
	opts := eseam.options
	if seam.timeout > 0 {
		opts = append(append([]engine.Option{}, eseam.options...), engine.WithTimeout(seam.timeout))
	}
	return engine.New(model, src, engine.Ports{
		Credentials: creds,
		Identities:  creds,
		Clock:       eseam.clock,
		Transport:   seamEngineTransport{base: base, rt: seam.client.Transport},
		Roots:       eseam.roots,
	}, opts...)
}

// seamEngineTransport sends to the seam base: it rewrites the request URL to the seam's base URL and round-trips with
// the seam client. The stub answers by path.
type seamEngineTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

// RoundTrip rewrites the request to the seam base and sends it with the seam client.
func (t seamEngineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.URL.Scheme = t.base.Scheme
	c.URL.Host = t.base.Host
	return t.rt.RoundTrip(c)
}

// specOperationID returns the engine operation of a catalog operation: the bundled spec prefixes the catalog ID with
// the kind, so documents.get runs as outline.documents.get.
func specOperationID(op domain.OperationID) domain.OperationID {
	return domain.OperationID(domain.KindOutline.String() + "." + string(op))
}

// Read runs one catalog operation against the wiki on the engine, restoring the collection or document not-found
// sentinel the retired Go gateway mapped.
func (g *outlineGateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	rec, err := g.engine.Read(ctx, specOperationID(op), args)
	return rec, translateOutlineNotFound(op, err)
}

// translateOutlineNotFound restores the not-found sentinel of an engine-served read, which the engine reports as a
// plain not found. Any other outcome passes through, so an unavailable method keeps its own failure.
func translateOutlineNotFound(op domain.OperationID, err error) error {
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return err
	}
	if collectionNotFoundReads[op] {
		return domain.ErrCollectionNotFound
	}
	if documentNotFoundReads[op] {
		return domain.ErrDocumentNotFound
	}
	return err
}

// isNotFound reports whether err is a plain not found, and not a sentinel of a missing collection or document.
func isNotFound(err error) bool {
	return err != nil && errors.Is(err, domain.ErrNotFound) &&
		!errors.Is(err, domain.ErrCollectionNotFound) && !errors.Is(err, domain.ErrDocumentNotFound)
}

// Write runs one change operation against the wiki on the engine.
func (g *outlineGateway) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	return g.engine.Write(ctx, specOperationID(op), args)
}

// ListCollections returns the collections of the source from the bundled-spec engine, which pages every page of
// the page-all rule and fails at the page cap.
func (g *outlineGateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	return g.engine.ListCollections(ctx)
}

// CollectionTree returns the documents of one collection from the engine's tree read of the bundled spec, mapped to
// the flat nodes the retired Go gateway answered: the id, title, and URL of each entry, with the parent ID of every
// child and the depth from 0. A missing collection keeps the collection-not-found sentinel of the tree read.
func (g *outlineGateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	rec, err := g.engine.Read(ctx, specOperationID(domain.OpCollectionsDocuments),
		domain.NewArgs(map[string]any{"collectionId": id.String()}))
	if err != nil {
		return nil, translateOutlineNotFound(domain.OpCollectionsDocuments, err)
	}
	return treeNodes(rec)
}

// treeNodes maps the items of a tree read to flat nodes. The parent of a top node is null, so its ParentID stays
// nil, and the depth counts from 0.
func treeNodes(rec domain.Record) ([]domain.TreeNode, error) {
	out := make([]domain.TreeNode, 0, len(rec.Items()))
	for _, item := range rec.Items() {
		node := domain.TreeNode{ID: item.Str("id"), Title: item.Str("title"), URL: item.Str("url")}
		if depth, ok := item.Get("depth"); ok {
			if d, isInt := depth.(int); isInt {
				node.Depth = d
			}
		}
		if parent, ok := item.Get("parentId"); ok {
			if p, isText := parent.(string); isText {
				node.ParentID = &p
			}
		}
		out = append(out, node)
	}
	return out, nil
}

// Document returns one document from the engine's get read of the bundled spec, with its Markdown body: the wrapped
// and unwrapped reply shapes map through the spec's fallback pointers, and a missing body maps through the declared
// export fallback. A missing document keeps the document-not-found sentinel of the get read.
func (g *outlineGateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	rec, err := g.engine.Read(ctx, specOperationID(domain.OpDocumentsGet),
		domain.NewArgs(map[string]any{"documentId": id.String()}))
	if err != nil {
		return domain.SourceDocument{}, translateOutlineNotFound(domain.OpDocumentsGet, err)
	}
	return domain.SourceDocument{
		ID:           rec.Str("id"),
		Title:        rec.Str("title"),
		URL:          rec.Str("url"),
		Markdown:     rec.Str("markdown"),
		CollectionID: rec.Str("collectionId"),
	}, nil
}
