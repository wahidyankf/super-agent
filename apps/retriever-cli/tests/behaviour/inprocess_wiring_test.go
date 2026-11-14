package behaviour

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/graphql"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	mcpadapter "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/mcp"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-jira/jira"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi/kiwi"
	"github.com/wahidyankf/super-agent/libs/retriever-connector-outline/outline"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/registry"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
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

// wireSources loads the catalog, builds the gateway registry, and returns the source service, as wireSources does in
// the composition root. A catalog error is returned, so only the source commands report it.
func (w *world) wireSources(paths configpaths.Paths, seams seamSet) (sources.Service, error) {
	if err := checkSeams(seams); err != nil {
		return sources.Service{}, err
	}
	creds, err := w.be.credentials(paths.EnvFile, w.getenv)
	if err != nil {
		return sources.Service{}, err
	}
	cat, err := w.be.loadCatalog(paths.Catalog, w.kinds)
	if err != nil {
		return sources.Service{}, err
	}
	resolver := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
			return wireOutline(src, creds, seams[domain.KindOutline], w.clock)
		},
		domain.KindKiwi: func(src domain.Source) (ports.SourceGateway, error) {
			return wireKiwi(src, creds, seams[domain.KindKiwi], w.clock)
		},
		domain.KindJira: func(src domain.Source) (ports.SourceGateway, error) {
			return wireJira(src, creds, seams[domain.KindJira], w.clock)
		},
	})
	return sources.Service{Catalog: cat, Gateways: resolver}, nil
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
		CatalogFlag: g.Catalog,
		EnvFileFlag: g.EnvFile,
		Getenv:      w.getenv,
		Getwd:       w.getwd,
		Exists:      w.be.regularFile,
	})
	if err != nil {
		return err
	}
	svc, err := w.wireSources(paths, seams)
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
