package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// goServedReads are the Outline reads the composite keeps on the Go gateway until the spec language can serve them.
// Reads not named here run on the bundled-spec engine. Pull request 17 removes this table with the retired module.
//
//   - collections.list declares no page parameters, so no spec operation can page it, and the page-cap scenario
//     needs the error the Go reader returns at the cap.
//   - documents.get shares one status for a missing document and a missing method, which no spec error mapping can
//     tell apart, and its wrapped and unwrapped reply shapes need two pointers.
//   - The comment reads map editor data the spec cannot compute, the list reads below map reply lists the spec
//     cannot read, and the integration reads compute address hosts the spec cannot compute.
var goServedReads = map[domain.OperationID]bool{
	domain.OpCollectionsList: true,
	domain.OpDocumentsGet:    true,

	domain.OpCommentsGet:  true,
	domain.OpCommentsList: true,

	domain.OpAPIKeysList:                 true,
	domain.OpAuthenticationProvidersGet:  true,
	domain.OpAuthenticationProvidersList: true,
	domain.OpIntegrationsGet:             true,
	domain.OpIntegrationsList:            true,
	domain.OpOAuthAuthenticationsList:    true,
	domain.OpOAuthClientsGet:             true,
	domain.OpOAuthClientsList:            true,
	domain.OpWebhookSubscriptionsList:    true,
}

// collectionNotFoundReads are the engine-served reads whose 404 names a missing collection, and documentNotFoundReads
// are the ones whose 404 names a missing document, as the Go gateway maps them. The engine reports both as not found,
// so the composite restores the sentinel the scenarios and the callers read. Any other engine failure passes through.
var (
	collectionNotFoundReads = map[domain.OperationID]bool{
		domain.OpCollectionsInfo:         true,
		domain.OpCollectionsDocuments:    true,
		domain.OpCollectionsMemberships:  true,
		domain.OpCollectionsGroupMembers: true,
	}
	documentNotFoundReads = map[domain.OperationID]bool{
		domain.OpDocumentsChildren:         true,
		domain.OpDocumentsInsights:         true,
		domain.OpDocumentsMemberships:      true,
		domain.OpDocumentsGroupMemberships: true,
		domain.OpDocumentsUsers:            true,
	}
)

// outlineComposite is the Outline gateway of pull request 16: one source served by the Go gateway and the
// bundled-spec engine together. Reads the spec serves run on the engine, every other operation runs on the Go
// gateway, and the legacy collection, tree, and document reads keep the Go gateway until pull request 17 retires it.
type outlineComposite struct {
	goGateway ports.SourceGateway
	goReader  ports.SourceReader
	goWriter  ports.SourceWriter
	engine    *engine.Engine
}

var (
	_ ports.SourceGateway = (*outlineComposite)(nil)
	_ ports.SourceReader  = (*outlineComposite)(nil)
	_ ports.SourceWriter  = (*outlineComposite)(nil)
)

// buildOutlineComposite returns the gateway of one Outline source: the Go gateway for the operations the spec does
// not serve yet, and the engine of the bundled outline.yml for the reads it serves. A source whose Go gateway cannot
// be built, or whose engine cannot, stops the source.
func buildOutlineComposite(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	eseam engineSeam) (ports.SourceGateway, error) {
	goGateway, err := buildOutline(src, creds, seam)
	if err != nil {
		return nil, err
	}
	eng, err := buildOutlineEngine(model, src, creds, seam, eseam)
	if err != nil {
		return nil, err
	}
	return &outlineComposite{goGateway: goGateway, goReader: goGateway, goWriter: goGateway, engine: eng}, nil
}

// buildOutlineEngine returns the engine of the bundled outline.yml for one source. The seam, when it is on, replaces
// only the connection base here too: the transport rewrites each request to the seam base and round-trips with the
// seam client, so the engine reaches the same stub the Go gateway reaches.
func buildOutlineEngine(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	eseam engineSeam) (*engine.Engine, error) {
	if !seam.on || seam.client == nil {
		return buildSpec(model, src, eseam, creds)
	}
	base, err := url.Parse(seam.base)
	if err != nil {
		return nil, err
	}
	return engine.New(model, src, engine.Ports{
		Credentials: creds,
		Identities:  creds,
		Clock:       eseam.clock,
		Transport:   seamEngineTransport{base: base, rt: seam.client.Transport},
		Roots:       eseam.roots,
	}, eseam.options...)
}

// seamEngineTransport sends to the seam base: it rewrites the request URL to the seam's base URL and round-trips with
// the seam client. The stub answers by path, as it does for the Go gateway.
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

// Read runs one catalog operation against the wiki. A read the bundled spec serves runs on the engine, and any other
// operation runs on the Go gateway.
func (g *outlineComposite) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	if goServedReads[op] {
		return g.goReader.Read(ctx, op, args)
	}
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

// Write runs one change operation against the wiki on the Go gateway, until pull request 17 ports the writes.
func (g *outlineComposite) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	return g.goWriter.Write(ctx, op, args)
}

// ListCollections returns the collections of the source from the Go gateway, until pull request 17 retires it.
func (g *outlineComposite) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	return g.goGateway.ListCollections(ctx)
}

// CollectionTree returns the documents of one collection from the Go gateway, until pull request 17 retires it.
func (g *outlineComposite) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	return g.goGateway.CollectionTree(ctx, id)
}

// Document returns one document from the Go gateway, until pull request 17 retires it.
func (g *outlineComposite) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	return g.goGateway.Document(ctx, id)
}
