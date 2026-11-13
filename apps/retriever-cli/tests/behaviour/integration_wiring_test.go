package behaviour

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/catalog"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/configpaths"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/envfile"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/graphql"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	mcpadapter "github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/mcp"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/jira"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/kiwi"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/out/markdownfiles"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/outline"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/registry"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/sources"
)

// The integration level mirrors the composition root in cmd/retriever (main.go and wiring.go) in process. The root is in
// package main, so it cannot be imported, and these functions repeat its wiring step for step. One difference is
// deliberate: the root reads secrets through cli.SecretEnv, which reads the process environment, while this level reads
// the scenario environment. The end-to-end level runs the real root, so that path is covered there.

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

// exportSink returns the sink that writes the exported Markdown files into the folder the --out flag names.
func exportSink(dir string) ports.ExportSink {
	return markdownfiles.New(dir)
}

// readSeam reads the seam of one kind through getenv. An invalid seam is an error.
func readSeam(kind domain.SourceKind, getenv func(string) string) (seamConfig, error) {
	switch kind {
	case domain.KindOutline:
		seam, err := outline.TestSeamFromEnv(getenv)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	case domain.KindKiwi:
		seam, err := kiwi.TestSeamFromEnv(getenv)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	case domain.KindJira:
		seam, err := jira.TestSeamFromEnv(getenv)
		if err != nil {
			return seamConfig{}, err
		}
		return seamConfig{base: seam.BaseURL, client: seam.Client, on: seam.Enabled, timeout: seam.Timeout}, nil
	}
	return seamConfig{}, fmt.Errorf("no test seam for the %s kind", kind)
}

// readSeams reads the seam of every kind. One bad seam fails the whole set, as it stops every command at startup.
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
func wireSources(paths configpaths.Paths, seams seamSet) (sources.Service, error) {
	if err := checkSeams(seams); err != nil {
		return sources.Service{}, err
	}
	creds, err := envfile.New(paths.EnvFile)
	if err != nil {
		return sources.Service{}, err
	}
	loader, err := catalog.New(paths.Catalog)
	if err != nil {
		return sources.Service{}, err
	}
	cat, err := loader.Load()
	if err != nil {
		return sources.Service{}, err
	}
	resolver := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
			return wireOutline(src, creds, seams[domain.KindOutline])
		},
		domain.KindKiwi: func(src domain.Source) (ports.SourceGateway, error) {
			return wireKiwi(src, creds, seams[domain.KindKiwi])
		},
		domain.KindJira: func(src domain.Source) (ports.SourceGateway, error) {
			return wireJira(src, creds, seams[domain.KindJira])
		},
	})
	return sources.Service{Catalog: cat, Gateways: resolver}, nil
}

// wireOutline returns the gateway of one Outline source. The seam, when it is on, replaces only the connection base.
func wireOutline(src domain.Source, creds ports.Credentials, seam seamConfig) (ports.SourceGateway, error) {
	if !seam.on {
		return outline.New(src, creds), nil
	}
	var opts []outline.Option
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
func wireKiwi(src domain.Source, creds ports.Credentials, seam seamConfig) (ports.SourceGateway, error) {
	if !seam.on {
		return kiwi.New(src, creds), nil
	}
	var opts []kiwi.Option
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
func wireJira(src domain.Source, secrets jira.Secrets, seam seamConfig) (ports.SourceGateway, error) {
	if !seam.on {
		return jira.New(src, secrets), nil
	}
	var opts []jira.Option
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
		Getwd:       os.Getwd,
		Exists:      regularFile,
	})
	if err != nil {
		return err
	}
	svc, err := wireSources(paths, seams)
	if err != nil {
		return err
	}
	deps.Service = svc
	deps.Serve = serveFunc(svc)
	deps.MCP = mcpFunc(svc)
	return nil
}

// serveFunc is the server of the serve command, as the composition root wires it: the GraphQL API over the sources,
// capped to read access. Both levels run graphql.Run, the same body the root runs.
func serveFunc(svc sources.Service) cli.ServeFunc {
	return func(ctx context.Context, port int, ready func(addr string)) error {
		return graphql.Run(ctx, svc.Capped(domain.AccessRead), port, slog.New(slog.DiscardHandler), ready)
	}
}

// mcpFunc is the server of the mcp command, as the composition root wires it: the sources capped to the ceiling the
// command names, offering the resources it names.
func mcpFunc(svc sources.Service) cli.MCPFunc {
	return func(ctx context.Context, ceiling domain.Access, resources []string, in io.Reader, out io.Writer) error {
		return mcpadapter.Runner(svc.Capped(ceiling), cli.Version, io.Discard, resources)(ctx, in, out)
	}
}

// protocolDeps returns the dependencies of a command that runs for a while, such as serve or mcp, in process. The
// process environment becomes the scenario environment, the given streams are attached, and the run context is the one
// the caller cancels to send a signal. A setup error is kept in SetupErr, as the composition root keeps it.
func (w *world) protocolDeps(ctx context.Context, stdin io.Reader, stdout io.Writer, argv []string) cli.Deps {
	w.adoptProcessEnv()
	deps := cli.Deps{
		Ports:   ports.Ports{},
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  io.Discard,
		Getenv:  w.getenv,
		Now:     time.Now,
		Sink:    exportSink,
		Context: ctx,
	}
	seams, err := readSeams(w.getenv)
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
	w.adoptProcessEnv()
	deps := w.deps(&out, &errb)
	if w.stdoutClosed {
		deps.Stdout = brokenPipe{}
	}
	seams, err := readSeams(w.getenv)
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
