package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/graphql"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	mcpadapter "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/mcp"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/connectorspec"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/jiraseam"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/outlineseam"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/operations"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-plugin"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
	"github.com/wahidyankf/super-agent/libs/retriever-spec"
)

// seamConfig is the loopback test seam for the connection base of one source kind. When it is off, each source's own
// endpoint is used.
type seamConfig struct {
	base    string
	client  *http.Client
	on      bool
	timeout time.Duration
}

// seamSet holds the test seam of each source kind: one seamConfig per kind. A kind with no entry has
// its seam off. The variables of one kind never reach another kind's gateway.
type seamSet map[domain.SourceKind]seamConfig

// seamKinds lists the kinds that have a test seam, in the order their variables are read. The Outline seam feeds
// the bundled-spec engine: the base rewrites engine requests to the stub, and the timeout bounds them.
var seamKinds = []domain.SourceKind{domain.KindOutline, domain.KindKiwi, domain.KindJira}

// kiwiSeamOptions returns the gateway options the Kiwi seam adds. Without a seam timeout it returns none.
func kiwiSeamOptions(seam seamConfig) []kiwi.Option {
	if seam.timeout <= 0 {
		return nil
	}
	return []kiwi.Option{kiwi.WithTimeout(seam.timeout)}
}

// sourceWiring is what the CLI takes from the composition root for the source commands, the protocol servers included.
type sourceWiring struct {
	service sources.Service
	serve   cli.ServeFunc
	mcp     cli.MCPFunc
}

// readSeam reads the test seam of one kind through getenv. An invalid seam is an error, never a fallback to a real
// host. Each kind reads only its own variables. The Outline and Jira seams feed the bundled-spec engine, now that no
// Go gateway reads them.
func readSeam(kind domain.SourceKind, getenv func(string) string) (seamConfig, error) {
	switch kind {
	case domain.KindOutline:
		seam, err := outlineseam.TestSeamFromEnvWithFiles(getenv, os.ReadFile)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	case domain.KindKiwi:
		seam, err := kiwi.TestSeamFromEnvWithFiles(getenv, os.ReadFile)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	case domain.KindJira:
		seam, err := jiraseam.TestSeamFromEnvWithFiles(getenv, os.ReadFile)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	}
	return seamConfig{}, fmt.Errorf("no test seam for the %s kind", kind)
}

// readSeams reads the test seam of every kind through getenv. One bad seam is an error, so it stops every command.
func readSeams(getenv func(string) string) (seamSet, error) {
	set := seamSet{}
	for _, kind := range seamKinds {
		seam, err := readSeam(kind, getenv)
		if err != nil {
			return nil, err
		}
		set[kind] = seam
	}
	return set, nil
}

// checkSeams checks each seam that is on, at setup, so a bad base fails the wiring and not the first source call.
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

// registeredKinds returns the kinds this composition root registers, which a catalog may name. The root registers the
// three built-in kinds and no other. A kind that another build registers is passed to wireSources the same way, as a
// value, so no package holds a registry.
func registeredKinds() domain.KindSet {
	return domain.BuiltinKinds()
}

// wireSources loads the sources catalog, builds the gateway registry, and returns the source service and the serve and
// MCP functions for the configuration paths the caller resolved. The run registers the bundled specs, as a run that
// loaded no connector folder does, so the Outline gateway is built. The catalog is read once here. A catalog error
// stops the sources commands, and the process still starts for every other command.
func wireSources(paths configpaths.Paths, seams seamSet, h host, kinds domain.KindSet) (sourceWiring, error) {
	set, err := withBundled(connectorspec.Set{
		Kinds:      kinds,
		Operations: operations.BuiltinCatalog(),
		Digests:    map[string]string{},
	}, nil)
	if err != nil {
		return sourceWiring{}, err
	}
	return wireSourcesWith(paths, seams, h, set)
}

// wireSourcesWith is wireSources for a run that loaded connector specs. The set supplies the kinds the catalog may name,
// the digests its spec pins are checked against, the operation rows the commands and tools are built from, and one engine
// builder for each spec kind.
func wireSourcesWith(paths configpaths.Paths, seams seamSet, h host, set connectorspec.Set) (sourceWiring, error) {
	if err := checkSeams(seams); err != nil {
		return sourceWiring{}, err
	}
	creds, err := h.openEnvFile(paths.EnvFile)
	if err != nil {
		return sourceWiring{}, err
	}
	loader, err := h.openCatalog(paths.Catalog, set.Kinds)
	if err != nil {
		return sourceWiring{}, err
	}
	cat, err := loader.WithSpecDigests(set.Digests).WithProfile(paths.Profile).Load()
	if err != nil {
		return sourceWiring{}, err
	}
	// The pins are known only now, so a POST read takes the read hints of a spec that every source pins.
	set = set.WithPinned(loader.PinnedKinds())
	builders := map[domain.SourceKind]registry.Builder{
		domain.KindKiwi: func(src domain.Source) (ports.SourceGateway, error) {
			return buildKiwi(src, creds, seams[domain.KindKiwi])
		},
	}
	maps.Copy(builders, specBuilders(set, h.engine, creds))
	// The bundled Outline spec runs on its engine gateway: every Outline operation runs on the bundled-spec engine,
	// with the not-found translation the retired Go gateway mapped.
	if model := bundledOutline(set); model != nil {
		builders[domain.KindOutline] = func(src domain.Source) (ports.SourceGateway, error) {
			return buildOutlineGateway(model, src, creds, seams[domain.KindOutline], h.engine)
		}
	}
	// The bundled Jira spec runs on its engine gateway: every Jira operation runs on the bundled-spec engine,
	// with the document-not-found translation the retired Go gateway mapped.
	if model := bundledJira(set); model != nil {
		builders[domain.KindJira] = func(src domain.Source) (ports.SourceGateway, error) {
			return buildJiraGateway(model, src, creds, seams[domain.KindJira], h.engine)
		}
	}
	// A source with a plugin block is served by its pinned plugin instead of its kind's gateway, whatever the kind.
	// A source without one keeps its kind's builder and starts no process (threat P2).
	for kind, built := range builders {
		inner := built
		builders[kind] = func(src domain.Source) (ports.SourceGateway, error) {
			if src.HasPlugin() {
				return buildPlugin(src, creds, creds, h.plugin)
			}
			return inner(src)
		}
	}
	resolver := registry.New(builders)
	ops := set.Operations
	svc := sources.Service{Catalog: cat, Gateways: resolver, Operations: &ops, Profile: paths.Profile}
	// The GraphQL server serves the reads of the catalog only: its schema has no mutation.
	return sourceWiring{
		service: svc,
		serve:   graphqlServe(svc, h.stderr, graphql.Listen),
		mcp:     mcpServe(svc, h.stderr),
	}, nil
}

// bundledOutline returns the bundled outline.yml model of the set, or nil when the set holds none. The merged set
// always holds it, because a user spec that shadows a bundled kind stops start-up.
func bundledOutline(set connectorspec.Set) *spec.Spec {
	for _, s := range set.Specs {
		if s.Model.Kind == domain.KindOutline.String() {
			return s.Model
		}
	}
	return nil
}

// bundledJira returns the bundled jira.yml model of the set, or nil when the set holds none. The merged set always
// holds it, because a user spec that shadows a bundled kind stops start-up.
func bundledJira(set connectorspec.Set) *spec.Spec {
	for _, s := range set.Specs {
		if s.Model.Kind == domain.KindJira.String() {
			return s.Model
		}
	}
	return nil
}

// buildPlugin returns the gateway of one plugin source: the pinned program serves the source through JSON-RPC,
// with the secrets the credentials and identities resolve. The seams run the real file and process calls in
// production, and fakes in a unit test.
func buildPlugin(src domain.Source, creds ports.Credentials, idents ports.Identities, s plugin.Seams) (ports.SourceGateway, error) {
	return &plugin.Gateway{Source: src, Credentials: creds, Identities: idents, Seams: s}, nil
}

// buildKiwi returns the gateway of one Kiwi source: the source's endpoint receives its key, and the seam, when it
// is on, replaces only the connection base.
func buildKiwi(src domain.Source, creds ports.Credentials, seam seamConfig) (ports.SourceGateway, error) {
	if !seam.on {
		return kiwi.New(src, creds), nil
	}
	gw, err := kiwi.NewWithBase(src, seam.base, seam.client, creds, kiwiSeamOptions(seam)...)
	if err != nil {
		return nil, err
	}
	return gw, nil
}

// graphqlServe returns the function the serve command runs. It opens the listener with listen, mounts the GraphQL handler,
// reports the bound address, and shuts down gracefully when ctx ends. The composition root passes graphql.Listen, the real
// loopback socket; a unit test passes an in-memory listener.
func graphqlServe(svc sources.Service, stderr io.Writer, listen graphql.ListenFunc) cli.ServeFunc {
	logger := slog.New(slog.NewTextHandler(stderr, nil))
	return func(ctx context.Context, port int, ready func(addr string)) error {
		return graphql.Run(ctx, svc, port, listen, logger, ready)
	}
}

// mcpServe returns the function the mcp command runs, offering the tools of the resources it names (nil for every
// resource). The server writes one log line per tool call to stderr, and standard output stays the protocol channel the
// caller passes in.
func mcpServe(svc sources.Service, stderr io.Writer) cli.MCPFunc {
	return func(ctx context.Context, resources []string, in io.Reader, out io.Writer) error {
		return mcpadapter.Runner(svc, cli.Version, stderr, resources)(ctx, in, out)
	}
}
