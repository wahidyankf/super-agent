package behaviour

import (
	"bytes"
	"context"
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
	"github.com/wahidyankf/super-agent/libs/retriever-connector-jira/jira"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-outline/outline"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-engine/engine"
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

// readSeam reads the seam of one kind through getenv. An invalid seam is an error.
func readSeam(kind domain.SourceKind, getenv func(string) string,
	readFile func(string) ([]byte, error)) (seamConfig, error) {
	switch kind {
	case domain.KindOutline:
		seam, err := outline.TestSeamFromEnvWithFiles(getenv, readFile)
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
		seam, err := jira.TestSeamFromEnvWithFiles(getenv, readFile)
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
		if err := outline.CheckBaseURL(seam.base); err != nil {
			return err
		}
	}
	if seam := seams[domain.KindKiwi]; seam.on {
		if err := kiwi.CheckBaseURL(seam.base); err != nil {
			return err
		}
	}
	if seam := seams[domain.KindJira]; seam.on {
		if err := jira.CheckBaseURL(seam.base); err != nil {
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
// Outline spec rides with the Go gateway in the composite, as the root wires it. A catalog error is returned, so only
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
	cat, err := w.be.loadCatalog(paths.Catalog, kinds, set.Digests)
	if err != nil {
		return sources.Service{}, err
	}
	builders := map[domain.SourceKind]registry.Builder{
		domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
			return wireOutline(src, creds, seams[domain.KindOutline], w.clock)
		},
		domain.KindKiwi: func(src domain.Source) (ports.SourceGateway, error) {
			return wireKiwi(src, creds, seams[domain.KindKiwi], w.clock)
		},
		domain.KindJira: func(src domain.Source) (ports.SourceGateway, error) {
			return wireJira(src, creds, seams[domain.KindJira], w.clock)
		},
	}
	for _, s := range set.Specs {
		builders[s.Kind] = w.specBuilder(rt, s.Model, creds)
	}
	// The bundled Outline spec rides with the Go gateway until pull request 17, as the root wires it: the composite
	// routes each operation to the engine or to the gateway, and nothing retires yet.
	if model := bundledOutlineModel(set); model != nil {
		seam := seams[domain.KindOutline]
		builders[domain.KindOutline] = func(src domain.Source) (ports.SourceGateway, error) {
			return w.buildOutlineComposite(model, src, creds, seam, rt)
		}
	}
	ops := set.Operations
	return sources.Service{Catalog: cat, Gateways: registry.New(builders), Operations: &ops}, nil
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

// goServedOutlineReads are the Outline reads the harness composite keeps on the Go gateway, mirroring goServedReads
// in the composition root. Reads not named here run on the bundled-spec engine, and nothing retires.
var goServedOutlineReads = map[domain.OperationID]bool{
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

// outlineNotFoundMaps mirrors the translation tables of the composition root: the engine reports every 404 as a plain
// not found, and the composite restores the collection and document sentinels the Go gateway maps.
var (
	outlineCollectionNotFoundReads = map[domain.OperationID]bool{
		domain.OpCollectionsInfo:         true,
		domain.OpCollectionsDocuments:    true,
		domain.OpCollectionsMemberships:  true,
		domain.OpCollectionsGroupMembers: true,
	}
	outlineDocumentNotFoundReads = map[domain.OperationID]bool{
		domain.OpDocumentsChildren:         true,
		domain.OpDocumentsInsights:         true,
		domain.OpDocumentsMemberships:      true,
		domain.OpDocumentsGroupMemberships: true,
		domain.OpDocumentsUsers:            true,
	}
)

// harnessOutlineComposite is the Outline gateway of a scenario: the Go gateway of the seam and the engine of the
// bundled outline.yml together, mirroring outlineComposite in the composition root. Reads the spec serves run on the
// engine over the seam transport, and every other operation runs on the Go gateway.
type harnessOutlineComposite struct {
	goGateway ports.SourceGateway
	goReader  ports.SourceReader
	goWriter  ports.SourceWriter
	engine    *engine.Engine
}

// buildOutlineComposite returns the gateway of one Outline source of a scenario, mirroring buildOutlineComposite in
// the composition root. The engine reaches the stub Outline server through the seam: the transport rewrites the
// request to the seam base and round-trips with the seam client, so the unit and integration levels need no socket of
// their own. A source whose Go gateway or engine cannot be built stops the scenario.
func (w *world) buildOutlineComposite(model *spec.Spec, src domain.Source, creds *envfile.Credentials, seam seamConfig,
	rt specRuntime) (ports.SourceGateway, error) {
	goGateway, err := wireOutline(src, creds, seam, w.clock)
	if err != nil {
		return nil, err
	}
	goReader, ok := goGateway.(ports.SourceReader)
	if !ok {
		return nil, domain.ErrUnsupportedOperation
	}
	goWriter, ok := goGateway.(ports.SourceWriter)
	if !ok {
		return nil, domain.ErrUnsupportedOperation
	}
	eng, err := engine.New(model, src, engine.Ports{
		Credentials: creds,
		Identities:  creds,
		Clock:       engineClock{clk: w.clock},
		Transport:   harnessEngineTransport(seam, rt),
	})
	if err != nil {
		return nil, err
	}
	return &harnessOutlineComposite{goGateway: goGateway, goReader: goReader, goWriter: goWriter, engine: eng}, nil
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
// seam client. The engine keeps its own base for signing nothing, and the stub answers by path, as it does for the
// Go gateway.
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

// Read runs one catalog operation of a scenario against the stub. A read the bundled spec serves runs on the engine,
// and any other operation runs on the Go gateway.
func (g *harnessOutlineComposite) Read(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	if goServedOutlineReads[op] {
		return g.goReader.Read(ctx, op, args)
	}
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

// Write runs one change operation of a scenario on the Go gateway, until pull request 17 ports the writes.
func (g *harnessOutlineComposite) Write(ctx context.Context, op domain.OperationID, args domain.Args) (domain.Record, error) {
	return g.goWriter.Write(ctx, op, args)
}

// ListCollections returns the collections of the source from the Go gateway, until pull request 17 retires it.
func (g *harnessOutlineComposite) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	return g.goGateway.ListCollections(ctx)
}

// CollectionTree returns the documents of one collection from the Go gateway, until pull request 17 retires it.
func (g *harnessOutlineComposite) CollectionTree(ctx context.Context, id domain.CollectionID) ([]domain.TreeNode, error) {
	return g.goGateway.CollectionTree(ctx, id)
}

// Document returns one document from the Go gateway, until pull request 17 retires it.
func (g *harnessOutlineComposite) Document(ctx context.Context, id domain.DocumentID) (domain.SourceDocument, error) {
	return g.goGateway.Document(ctx, id)
}

// wireOutline returns the gateway of one Outline source. The seam, when it is on, replaces only the connection base.
func wireOutline(src domain.Source, creds ports.Credentials, seam seamConfig, clk *fakeClock) (ports.SourceGateway, error) {
	opts := []outline.Option{outline.WithClock(clk.Now), outline.WithSleep(clk.Sleep)}
	if !seam.on {
		return outline.New(src, creds, opts...), nil
	}
	if seam.timeout > 0 {
		opts = append(opts, outline.WithTimeout(seam.timeout))
	}
	gw, err := outline.NewWithBase(src, seam.base, seam.client, creds, opts...)
	if err != nil {
		return nil, err
	}
	return gw, nil
}

// wireKiwi returns the gateway of one Kiwi source, under the same rule as wireOutline.
func wireKiwi(src domain.Source, creds ports.Credentials, seam seamConfig, clk *fakeClock) (ports.SourceGateway, error) {
	opts := []kiwi.Option{kiwi.WithClock(clk.Now), kiwi.WithSleep(clk.Sleep)}
	if !seam.on {
		return kiwi.New(src, creds, opts...), nil
	}
	if seam.timeout > 0 {
		opts = append(opts, kiwi.WithTimeout(seam.timeout))
	}
	gw, err := kiwi.NewWithBase(src, seam.base, seam.client, creds, opts...)
	if err != nil {
		return nil, err
	}
	return gw, nil
}

// wireJira returns the gateway of one Jira source, under the same rule as wireOutline. The credentials adapter
// provides both the token and the account e-mail address.
func wireJira(src domain.Source, secrets jira.Secrets, seam seamConfig, clk *fakeClock) (ports.SourceGateway, error) {
	opts := []jira.Option{jira.WithClock(clk.Now), jira.WithSleep(clk.Sleep)}
	if !seam.on {
		return jira.New(src, secrets, opts...), nil
	}
	if seam.timeout > 0 {
		opts = append(opts, jira.WithTimeout(seam.timeout))
	}
	gw, err := jira.NewWithBase(src, seam.base, seam.client, secrets, opts...)
	if err != nil {
		return nil, err
	}
	return gw, nil
}

// attachSources resolves the configuration paths and wires the sources into deps, as attachSourcesFrom does in the
// composition root. It also wires the serve and mcp servers over the same sources. The returned error becomes the setup
// error of the CLI.
func (w *world) attachSources(deps *cli.Deps, g cli.Globals, seams seamSet) error {
	paths, err := configpaths.Resolve(configpaths.Inputs{
		CatalogFlag:    g.Catalog,
		EnvFileFlag:    g.EnvFile,
		ConnectorsFlag: g.ConnectorsDir,
		Getenv:         w.getenv,
		Getwd:          w.getwd,
		Exists:         w.be.regularFile,
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
