package behaviour

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/graphql"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	mcpadapter "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/mcp"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/jiraseam"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/outlineseam"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
	"github.com/wahidyankf/super-agent/libs/retriever-plugin"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/specfolder"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// The in-process levels mirror the composition root in cmd/retriever (main.go and wiring.go). The root is in package
// main, so it cannot be imported, and these functions repeat its wiring step for step. The files, the credentials, the
// catalog, the export sink, and the working directory come from the level's backend: the real disk at the integration
// level, memory at the unit level. The end-to-end level runs the real root as the built binary.

// seamConfig is the loopback test seam of one source kind, as the composition root holds it.
type seamConfig struct {
	base    string
	client  *http.Client
	on      bool
	timeout time.Duration
}

// seamSet holds the seam of every kind. A kind's variables never reach another kind's gateway.
type seamSet map[domain.SourceKind]seamConfig

// seamKinds lists the kinds that have a seam, in the order their variables are read.
var seamKinds = []domain.SourceKind{domain.KindOutline, domain.KindKiwi, domain.KindJira}

// readSeam reads the seam of one kind through getenv. An invalid seam is an error. The Outline seam feeds the
// bundled-spec engine, now that no Go gateway reads them.
func readSeam(kind domain.SourceKind, getenv func(string) string,
	readFile func(string) ([]byte, error)) (seamConfig, error) {
	switch kind {
	case domain.KindOutline:
		seam, err := outlineseam.TestSeamFromEnvWithFiles(getenv, readFile)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	case domain.KindKiwi:
		seam, err := kiwi.TestSeamFromEnvWithFiles(getenv, readFile)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	case domain.KindJira:
		seam, err := jiraseam.TestSeamFromEnvWithFiles(getenv, readFile)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	}
	return seamConfig{}, fmt.Errorf("no test seam for the %s kind", kind)
}

// readSeams reads the seam of every kind. One bad seam fails the whole set, as it stops every command at startup.
func readSeams(getenv func(string) string, readFile func(string) ([]byte, error)) (seamSet, error) {
	set := seamSet{}
	for _, kind := range seamKinds {
		seam, err := readSeam(kind, getenv, readFile)
		if err != nil {
			return nil, err
		}
		set[kind] = seam
	}
	return set, nil
}

// checkSeams checks each seam that is on, so a bad base fails the wiring and not the first source call.
func checkSeams(seams seamSet) error {
	if seam := seams[domain.KindOutline]; seam.on {
		if err := outlineseam.CheckBaseURL(seam.base); err != nil {
			return err
		}
	}
	if seam := seams[domain.KindKiwi]; seam.on {
		if err := kiwi.CheckBaseURL(seam.base); err != nil {
			return err
		}
	}
	if seam := seams[domain.KindJira]; seam.on {
		if err := jiraseam.CheckBaseURL(seam.base); err != nil {
			return err
		}
	}
	return nil
}

// harnessFolderError is the failure of a connectors folder that the run cannot use. It mirrors the composition root's
// error: the same stable code, and a text that names no path.
type harnessFolderError struct {
	rule string
}

// Error returns the fixed text of the failure.
func (e harnessFolderError) Error() string { return "the connectors folder " + e.rule }

// ErrorCode returns the code that the composition root gives a folder the run cannot use.
func (harnessFolderError) ErrorCode() string { return "retriever.connector.folder_unavailable" }

// loadConnectors reads and validates the spec files of the connectors folder dir, and returns the set the run registers. It
// mirrors loadConnectors in the composition root: an empty dir loads nothing, a missing default folder loads nothing, and a
// folder that the flag named must exist and be readable.
func (w *world) loadConnectors(dir string, named bool) (connectorspec.Set, error) {
	if dir == "" {
		return connectorspec.Load(nil)
	}
	files, err := specfolder.Read(w.be.connectorsFolder(dir), spec.MaxBytes)
	var file specfolder.FileError
	switch {
	case errors.Is(err, specfolder.ErrNoFolder) && !named:
		return connectorspec.Load(nil)
	case errors.Is(err, specfolder.ErrNoFolder):
		return connectorspec.Set{}, harnessFolderError{rule: "does not exist"}
	case errors.As(err, &file):
		return connectorspec.Set{}, &connectorspec.Error{File: file.Name, Rule: file.Rule}
	case err != nil:
		return connectorspec.Set{}, harnessFolderError{rule: "cannot be read"}
	}
	return connectorspec.Load(files)
}

// wireSources loads the catalog, builds the gateway registry, and returns the source service, as wireSources does in
// the composition root. The spec sources of the set are built from its one engine builder each, and the bundled
// Outline spec runs on its engine gateway, as the root wires it. A catalog error is returned, so only
// the source commands report it.
func (w *world) wireSources(paths configpaths.Paths, seams seamSet, set connectorspec.Set, userSpecs bool) (sources.Service, error) {
	if err := checkSeams(seams); err != nil {
		return sources.Service{}, err
	}
	creds, err := w.be.credentials(paths.EnvFile, w.getenv)
	if err != nil {
		return sources.Service{}, err
	}
	rt, err := w.specRuntimeFor(set)
	if err != nil {
		return sources.Service{}, err
	}
	// The bundled specs of the merged set never change the registered kinds: only a user spec does, as the set held
	// before the merge.
	kinds := w.kinds
	if userSpecs {
		kinds = set.Kinds
	}
	cat, err := w.be.loadCatalog(paths.Catalog, kinds, set.Digests, paths.Profile)
	if err != nil {
		return sources.Service{}, err
	}
	builders := map[domain.SourceKind]registry.Builder{
		domain.KindKiwi: func(src domain.Source) (ports.SourceGateway, error) {
			return wireKiwi(src, creds, seams[domain.KindKiwi], w.clock)
		},
	}
	for _, s := range set.Specs {
		builders[s.Kind] = w.specBuilder(rt, s.Model, creds)
	}
	// The bundled Outline spec runs on its engine gateway, as the root wires it: every Outline operation runs on
	// the bundled-spec engine, with the not-found translation the retired Go gateway mapped.
	if model := bundledOutlineModel(set); model != nil {
		seam := seams[domain.KindOutline]
		builders[domain.KindOutline] = func(src domain.Source) (ports.SourceGateway, error) {
			return w.buildOutlineGateway(model, src, creds, seam, rt)
		}
	}
	// The bundled Jira spec runs on its engine gateway, as the root wires it: every Jira operation runs on
	// the bundled-spec engine, with the document-not-found translation the retired Go gateway mapped.
	if model := bundledJiraModel(set); model != nil {
		seam := seams[domain.KindJira]
		builders[domain.KindJira] = func(src domain.Source) (ports.SourceGateway, error) {
			return w.buildJiraGateway(model, src, creds, seam, rt)
		}
	}
	// A source with a plugin block is served by its pinned plugin instead of its kind's gateway, whatever the kind, as the
	// root wires it. The seams are this level's: fakes at the unit level, the production seams at the disk levels.
	plug := w.be.pluginSeams(w)
	for kind, built := range builders {
		inner := built
		builders[kind] = func(src domain.Source) (ports.SourceGateway, error) {
			if src.HasPlugin() {
				return buildPluginGateway(src, creds, plug)
			}
			return inner(src)
		}
	}
	ops := set.Operations
	return sources.Service{Catalog: cat, Gateways: registry.New(builders), Operations: &ops, Profile: paths.Profile}, nil
}

// buildPluginGateway returns the gateway of one plugin source, mirroring buildPlugin in the composition root: the pinned
// program serves the source, with the secrets that the credentials and identities resolve.
func buildPluginGateway(src domain.Source, creds *envfile.Credentials, s plugin.Seams) (ports.SourceGateway, error) {
	return &plugin.Gateway{Source: src, Credentials: creds, Identities: creds, Seams: s}, nil
}

// bundledOutlineModel returns the bundled outline.yml model of the set, or nil when the set holds none. It mirrors
// bundledOutline in the composition root, which package main holds and this package cannot import.
func bundledOutlineModel(set connectorspec.Set) *spec.Spec {
	for _, s := range set.Specs {
		if s.Model.Kind == domain.KindOutline.String() {
			return s.Model
		}
	}
	return nil
}

// outlineNotFoundReads mirrors the translation tables of the composition root: the engine reports every 404 as a
// plain not found, and the gateway restores the collection and document sentinels the retired Go gateway mapped.
var (
	outlineCollectionNotFoundReads = map[domain.OperationID]bool{
		domain.OpCollectionsInfo:         true,
		domain.OpCollectionsDocuments:    true,
		domain.OpCollectionsMemberships:  true,
		domain.OpCollectionsGroupMembers: true,
	}
	outlineDocumentNotFoundReads = map[domain.OperationID]bool{
		domain.OpDocumentsGet:              true,
		domain.OpDocumentsChildren:         true,
		domain.OpDocumentsInsights:         true,
		domain.OpDocumentsMemberships:      true,
		domain.OpDocumentsGroupMemberships: true,
		domain.OpDocumentsUsers:            true,
	}
)

// harnessOutlineGateway is the Outline gateway of a scenario: the engine of the bundled outline.yml, mirroring
// outlineGateway in the composition root. Every Outline operation runs on the engine over the seam transport, and the
// gateway restores the not-found sentinels the retired Go gateway mapped.
type harnessOutlineGateway struct {
	engine *engine.Engine
}

// buildOutlineGateway returns the gateway of one Outline source of a scenario, mirroring buildOutlineGateway in
// the composition root. The engine reaches the stub Outline server through the seam: the transport rewrites the
// request to the seam base and round-trips with the seam client, so the unit and integration levels need no socket of
// their own. A source whose engine cannot be built stops the scenario.
func (w *world) buildOutlineGateway(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	rt specRuntime) (ports.SourceGateway, error) {
	var opts []engine.Option
	if seam.timeout > 0 {
		opts = append(opts, engine.WithTimeout(seam.timeout))
	}
	eng, err := engine.New(model, src, engine.Ports{
		Credentials: creds,
		Identities:  creds,
		Clock:       engineClock{clk: w.clock},
		Transport:   harnessEngineTransport(seam, rt),
	}, opts...)
	if err != nil {
		return nil, err
	}
	return &harnessOutlineGateway{engine: eng}, nil
}

// harnessEngineTransport returns the transport of the scenario's Outline engine: the seam transport when the seam is
// on, which reaches the stub, or the spec runtime's transport otherwise, which serves no Outline exchange.
func harnessEngineTransport(seam seamConfig, rt specRuntime) http.RoundTripper {
	if !seam.on || seam.client == nil {
		return rt.transport
	}
	base, err := url.Parse(seam.base)
	if err != nil {
		return rt.transport
	}
	return seamTransport{base: base, rt: seam.client.Transport}
}

// seamTransport sends to the seam base: it rewrites the request URL to the seam's base URL and round-trips with the
// seam client. The engine keeps its own base for signing nothing, and the stub answers by path.
type seamTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

// RoundTrip rewrites the request to the seam base and sends it with the seam client.
func (t seamTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.URL.Scheme = t.base.Scheme
	c.URL.Host = t.base.Host
	return t.rt.RoundTrip(c)
}

// outlineSpecOperationID returns the engine operation of a catalog operation, mirroring specOperationID in the
// composition root.
func outlineSpecOperationID(op domain.OperationID) domain.OperationID {
	return domain.OperationID(domain.KindOutline.String() + "." + string(op))
}

// Read runs one catalog operation of a scenario against the stub on the engine, restoring the collection or
// document not-found sentinel.
func (g *harnessOutlineGateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	rec, err := g.engine.Read(ctx, outlineSpecOperationID(op), args)
	return rec, translateOutlineNotFound(op, err)
}

// translateOutlineNotFound restores the not-found sentinel of an engine-served read, mirroring the composition root.
func translateOutlineNotFound(op domain.OperationID, err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrCollectionNotFound) || errors.Is(err, domain.ErrDocumentNotFound) {
		return err
	}
	if outlineCollectionNotFoundReads[op] {
		return domain.ErrCollectionNotFound
	}
	if outlineDocumentNotFoundReads[op] {
		return domain.ErrDocumentNotFound
	}
	return err
}

// Write runs one change operation of a scenario against the stub on the engine.
func (g *harnessOutlineGateway) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	rec, err := g.engine.Write(ctx, outlineSpecOperationID(op), args)
	return rec, err
}

// ListCollections returns the collections of the source from the bundled-spec engine, mirroring the composition
// root: the page-all rule pages every page and fails at the page cap.
func (g *harnessOutlineGateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	return g.engine.ListCollections(ctx)
}

// CollectionTree returns the documents of one collection from the engine's tree read, mirroring CollectionTree in
// the composition root: the flat nodes with the parent ID of every child and the depth from 0. A missing collection
// keeps the collection-not-found sentinel of the tree read.
func (g *harnessOutlineGateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	rec, err := g.engine.Read(ctx, outlineSpecOperationID(domain.OpCollectionsDocuments),
		domain.NewArgs(map[string]any{"collectionId": id.String()}))
	if err != nil {
		return nil, translateOutlineNotFound(domain.OpCollectionsDocuments, err)
	}
	return harnessTreeNodes(rec)
}

// harnessTreeNodes maps the items of a tree read to flat nodes, mirroring treeNodes in the composition root. The
// parent of a top node is null, so its ParentID stays nil, and the depth counts from 0.
func harnessTreeNodes(rec domain.Record) ([]domain.TreeNode, error) {
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

// Document returns one document from the engine's get read, mirroring Document in the composition root, with its
// Markdown body. A missing document keeps the document-not-found sentinel of the get read.
func (g *harnessOutlineGateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	rec, err := g.engine.Read(ctx, outlineSpecOperationID(domain.OpDocumentsGet),
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

// wireKiwi returns the gateway of one Kiwi source: the seam, when it is on, replaces only the connection base.
func wireKiwi(src domain.Source, creds ports.Credentials, seam seamConfig, clk *fakeClock) (ports.SourceGateway, error) {
	opts := []kiwi.Option{kiwi.WithClock(clk.Now), kiwi.WithSleep(clk.Sleep)}
	if !seam.on {
		return kiwi.New(src, creds, opts...), nil
	}
	if seam.timeout > 0 {
		opts = append(opts, kiwi.WithTimeout(seam.timeout))
	}
	opts = append(opts, kiwi.WithLimit(engineClock{clk: clk}.WithTimeout))
	gw, err := kiwi.NewWithBase(src, seam.base, seam.client, creds, opts...)
	if err != nil {
		return nil, err
	}
	return gw, nil
}

// bundledJiraModel returns the bundled jira.yml model of the set, or nil when the set holds none. It mirrors
// bundledJira in the composition root, which package main holds and this package cannot import.
func bundledJiraModel(set connectorspec.Set) *spec.Spec {
	for _, s := range set.Specs {
		if s.Model.Kind == domain.KindJira.String() {
			return s.Model
		}
	}
	return nil
}

// jiraDocumentNotFoundReads mirrors the translation table of the composition root: the engine reports every 404 as a
// plain not found, and the gateway restores the document sentinel the retired Go gateway mapped.
var jiraDocumentNotFoundReads = map[domain.OperationID]bool{
	domain.OpJiraIssuesGet:    true,
	domain.OpJiraCommentsList: true,
}

// jiraSpecOperations mirrors the operation map of the composition root: each catalog operation runs as the bundled
// operation the map names. A spec operation never repeats a catalog row name, as the root documents.
var jiraSpecOperations = map[domain.OperationID]domain.OperationID{
	domain.OpJiraWhoami:       "jira.auth.whoami",
	domain.OpJiraProjectsList: "jira.projects.page",
	domain.OpJiraIssuesSearch: "jira.issues.query",
	domain.OpJiraIssuesGet:    "jira.issues.read",
	domain.OpJiraCommentsList: "jira.comments.page",
}

// harnessJiraGateway is the Jira gateway of a scenario: the engine of the bundled jira.yml, mirroring jiraGateway
// in the composition root. Every Jira operation runs on the engine over the seam transport, and the gateway restores
// the document-not-found sentinel the retired Go gateway mapped.
type harnessJiraGateway struct {
	engine *engine.Engine
}

// buildJiraGateway returns the gateway of one Jira source of a scenario, mirroring buildJiraGateway in the
// composition root. The engine reaches the stub Jira server through the seam: the transport rewrites the request to
// the seam base and round-trips with the seam client, so the unit and integration levels need no socket of their
// own. A source whose engine cannot be built stops the scenario.
func (w *world) buildJiraGateway(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	rt specRuntime) (ports.SourceGateway, error) {
	var opts []engine.Option
	if seam.timeout > 0 {
		opts = append(opts, engine.WithTimeout(seam.timeout))
	}
	transport := harnessEngineTransport(seam, rt)
	if seam.on && seam.client != nil {
		base, err := url.Parse(seam.base)
		if err != nil {
			return nil, err
		}
		if err := jiraseam.CheckBaseURL(seam.base); err != nil {
			return nil, err
		}
		transport = seamTransport{base: base, rt: hardenHarnessSeamTransport(seam.client.Transport, src.Endpoint().Host())}
	}
	eng, err := engine.New(model, src, engine.Ports{
		Credentials: creds,
		Identities:  creds,
		Clock:       engineClock{clk: w.clock},
		Network:     rt.network,
		Transport:   transport,
	}, opts...)
	if err != nil {
		return nil, err
	}
	return &harnessJiraGateway{engine: eng}, nil
}

// hardenHarnessSeamTransport returns a copy of the seam transport that sends the endpoint host as the TLS server
// name, mirroring hardenSeamTransport in the composition root, which package main holds and this package cannot
// import.
func hardenHarnessSeamTransport(rt http.RoundTripper, host string) http.RoundTripper {
	tr, ok := rt.(*http.Transport)
	if !ok {
		return rt
	}
	clone := tr.Clone()
	if clone.TLSClientConfig == nil {
		clone.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		clone.TLSClientConfig = clone.TLSClientConfig.Clone()
	}
	clone.TLSClientConfig.ServerName = host
	return clone
}

// Read runs one catalog operation of a scenario against the stub on the engine, restoring the document sentinel
// and setting the search page, mirroring Read in the composition root.
func (g *harnessJiraGateway) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	specOp, ok := jiraSpecOperations[op]
	if !ok {
		return domain.Record{}, domain.ErrUnsupportedOperation
	}
	rec, err := g.engine.Read(ctx, specOp, args)
	if err != nil {
		return domain.Record{}, restoreHarnessJiraError(op, err)
	}
	if op == domain.OpJiraIssuesSearch {
		rec = withHarnessSearchPage(rec)
	}
	return rec, nil
}

// translateJiraNotFound restores the document-not-found sentinel of an engine-served read, mirroring the composition
// root.
func translateJiraNotFound(op domain.OperationID, err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrCollectionNotFound) || errors.Is(err, domain.ErrDocumentNotFound) {
		return err
	}
	if jiraDocumentNotFoundReads[op] {
		return domain.ErrDocumentNotFound
	}
	return err
}

// restoreHarnessJiraError restores the bare sentinel of an unreachable source, mirroring restoreJiraError in the
// composition root. Every other error keeps the document translation of the operation.
func restoreHarnessJiraError(op domain.OperationID, err error) error {
	if errors.Is(err, domain.ErrUnreachable) {
		return domain.ErrUnreachable
	}
	return translateJiraNotFound(op, err)
}

// withHarnessSearchPage sets the page of an issue search from the truncated flag of its items, mirroring
// withSearchPage in the composition root.
func withHarnessSearchPage(rec domain.Record) domain.Record {
	items := rec.Items()
	more := false
	if len(items) > 0 {
		if flag, ok := items[0].Get("truncated"); ok {
			more, _ = flag.(bool)
		}
	}
	return rec.WithPage(domain.Page{Paged: true, HasMore: more, NextOffset: len(items)})
}

// ListCollections returns the projects of the source from the bundled-spec engine, mirroring ListCollections in
// the composition root: the spec maps the raw project key, and the gateway prefixes it.
func (g *harnessJiraGateway) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	rec, err := g.engine.Read(ctx, "jira.projects.all", domain.NewArgs(nil))
	if err != nil {
		return nil, restoreHarnessJiraError(domain.OpJiraProjectsList, err)
	}
	out := make([]domain.Collection, 0, len(rec.Items()))
	for _, item := range rec.Items() {
		id, err := jiraseam.ProjectCollectionID(item.Str("id"))
		if err != nil {
			return nil, domain.ErrUnexpectedResponse
		}
		out = append(out, domain.Collection{ID: id.String(), Name: item.Str("name")})
	}
	return out, nil
}

// CollectionTree returns the issues of one project from the engine's cursor walk of the bundled spec, mirroring
// CollectionTree in the composition root: the flat nodes with the parent ID of every child and the depth from 0.
// A project the server refuses keeps the collection-not-found sentinel of the tree walk.
func (g *harnessJiraGateway) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	key, err := jiraseam.ProjectKey(id)
	if err != nil {
		return nil, err
	}
	rec, err := g.engine.Read(ctx, "jira.issues.tree",
		domain.NewArgs(map[string]any{"jql": `project = "` + key + `" ORDER BY key ASC`}))
	if err != nil {
		if errors.Is(err, domain.ErrBadRequest) {
			return nil, domain.ErrCollectionNotFound
		}
		return nil, restoreHarnessJiraError(domain.OpJiraIssuesSearch, err)
	}
	return harnessJiraTreeNodes(rec)
}

// harnessJiraTreeNodes maps the issues of the tree walk to flat nodes, mirroring jiraTreeNodes in the composition
// root. The parent of a top node is null, so its ParentID stays nil, and the depth counts from 0.
func harnessJiraTreeNodes(rec domain.Record) ([]domain.TreeNode, error) {
	items := rec.Items()
	index := make(map[string]int, len(items))
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = item.Str("key")
		index[keys[i]] = i
	}
	parentOf := make([]int, len(items))
	nodes := make([]domain.TreeNode, len(items))
	for i, item := range items {
		node := domain.TreeNode{
			ID:    "issue-" + keys[i],
			Title: item.Str("title"),
			URL:   item.Str("url"),
		}
		parentOf[i] = -1
		if parent, ok := item.Get("parent"); ok {
			if text, isText := parent.(string); isText && text != "" {
				if _, err := jiraseam.ParseIssueKey(text); err == nil {
					if j, ok := index[text]; ok {
						parentOf[i] = j
						issue := "issue-" + text
						node.ParentID = &issue
					}
				}
			}
		}
		nodes[i] = node
	}
	for i := range nodes {
		depth := 0
		for j := parentOf[i]; j >= 0; j = parentOf[j] {
			depth++
			if depth > len(nodes) {
				return nil, domain.ErrUnexpectedResponse
			}
		}
		nodes[i].Depth = depth
	}
	return nodes, nil
}

// Document returns one issue from the engine's get read, mirroring Document in the composition root, with its
// Markdown body. A missing issue keeps the document-not-found sentinel of the get read.
func (g *harnessJiraGateway) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	key, err := jiraseam.IssueKey(id)
	if err != nil {
		return domain.SourceDocument{}, err
	}
	rec, err := g.engine.Read(ctx, jiraSpecOperations[domain.OpJiraIssuesGet],
		domain.NewArgs(map[string]any{"issueKey": key}))
	if err != nil {
		return domain.SourceDocument{}, restoreHarnessJiraError(domain.OpJiraIssuesGet, err)
	}
	return domain.SourceDocument{
		ID:           rec.Str("id"),
		Title:        rec.Str("title"),
		URL:          rec.Str("url"),
		Markdown:     rec.Str("markdown"),
		CollectionID: rec.Str("collectionId"),
	}, nil
}

// attachSources resolves the configuration paths and wires the sources into deps, as attachSourcesFrom does in the
// composition root. It also wires the serve and mcp servers over the same sources. The returned error becomes the setup
// error of the CLI.
func (w *world) attachSources(deps *cli.Deps, g cli.Globals, seams seamSet) error {
	paths, err := configpaths.Resolve(configpaths.Inputs{
		CatalogFlag:    g.Catalog,
		EnvFileFlag:    g.EnvFile,
		ConnectorsFlag: g.ConnectorsDir,
		ProfileFlag:    g.Profile,
		Getenv:         w.getenv,
		Getwd:          w.getwd,
		Exists:         w.be.regularFile,
		IsDir:          w.be.dirExists,
		IsSymlink:      w.be.isSymlink,
	})
	if err != nil {
		return err
	}
	userSet, err := w.loadConnectors(paths.Connectors, g.ConnectorsDir != "")
	if err != nil {
		return err
	}
	// The bundled specs merge with the user-folder set, as withBundled merges them in the composition root. A user
	// spec that shadows a bundled kind stops the run here, before any source is built.
	bundled, err := connectorspec.LoadBundled()
	if err != nil {
		return err
	}
	set, err := connectorspec.Merge(userSet, bundled)
	if err != nil {
		return err
	}
	// The spec commands are registered before the wiring, as the composition root registers them, so a refused catalog
	// still reports its failure through the command that names the spec kind.
	deps.Specs = cli.Specs{Catalog: set.Operations, Rows: set.CLIRows}
	svc, err := w.wireSources(paths, seams, set, len(userSet.Specs) > 0)
	if err != nil {
		return err
	}
	deps.Service = svc
	deps.Serve = serveFunc(svc, w.be.listenPort)
	deps.MCP = mcpFunc(svc)
	return nil
}

// serveFunc is the server of the serve command, as the composition root wires it: the GraphQL API over the sources.
// Both levels run graphql.Run, the same body the root runs.
func serveFunc(svc sources.Service, listen graphql.ListenFunc) cli.ServeFunc {
	return func(ctx context.Context, port int, ready func(addr string)) error {
		return graphql.Run(ctx, svc, port, listen, slog.New(slog.DiscardHandler), ready)
	}
}

// mcpFunc is the server of the mcp command, as the composition root wires it: the sources, offering the resources the
// command names.
func mcpFunc(svc sources.Service) cli.MCPFunc {
	return func(ctx context.Context, resources []string, in io.Reader, out io.Writer) error {
		return mcpadapter.Runner(svc, cli.Version, io.Discard, resources)(ctx, in, out)
	}
}

// protocolDeps returns the dependencies of a command that runs for a while, such as serve or mcp, in process. The
// backend makes the process environment the scenario environment where the level uses it, the given streams are
// attached, and the run context is the one the caller cancels to send a signal. A setup error is kept in SetupErr, as
// the composition root keeps it.
func (w *world) protocolDeps(ctx context.Context, stdin io.Reader, stdout io.Writer, argv []string) cli.Deps {
	w.be.adoptEnv(w.env)
	deps := cli.Deps{
		Ports:   ports.Ports{},
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  io.Discard,
		Getenv:  w.getenv,
		Now:     w.clock.Now,
		Notify:  noSignalNotify,
		Stop:    noSignalStop,
		Sink:    w.be.sink,
		Context: ctx,
	}
	seams, err := w.readSeamSet()
	if err != nil {
		deps.SetupErr = err
		return deps
	}
	globals, _ := cli.ParseGlobals(argv)
	deps.SetupErr = w.attachSources(&deps, globals, seams)
	return deps
}

// runIntegration runs one command line in process with the real adapters. A seam that is invalid stops the command at
// startup, as it does in the composition root, and prints the same message.
func (w *world) runIntegration(argv []string) error {
	var out, errb bytes.Buffer
	w.be.adoptEnv(w.env)
	deps := w.deps(&out, &errb)
	if w.stdoutClosed {
		deps.Stdout = brokenPipe{}
	}
	seams, err := w.readSeamSet()
	if err != nil {
		fmt.Fprintf(&errb, "retriever: %v\n", err)
		w.record(cli.StartupStatus(argv), &out, &errb)
		return nil
	}
	globals, _ := cli.ParseGlobals(argv)
	deps.SetupErr = w.attachSources(&deps, globals, seams)
	w.record(cli.Run(argv, deps), &out, &errb)
	return nil
}

// readSeamSet reads the seam of every kind through the scenario environment. Each seam client is replaced by the level's
// client, so a gateway dials only what the level holds: the in-memory network at the unit level, the real network at the
// disk levels.
func (w *world) readSeamSet() (seamSet, error) {
	seams, err := readSeams(w.getenv, w.be.readFile)
	if err != nil {
		return nil, err
	}
	for kind, seam := range seams {
		if seam.client != nil {
			seam.client = w.be.seamClient(seam.client)
			seams[kind] = seam
		}
	}
	return seams, nil
}
