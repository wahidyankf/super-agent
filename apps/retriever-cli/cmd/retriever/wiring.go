package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
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

// seamKinds lists the kinds that have a test seam, in the order their variables are read.
var seamKinds = []domain.SourceKind{domain.KindOutline, domain.KindKiwi, domain.KindJira}

// seamOptions returns the gateway options the Outline seam adds. Without a seam timeout it returns none,
// so the gateway keeps its production default.
func seamOptions(seam seamConfig) []outline.Option {
	if seam.timeout <= 0 {
		return nil
	}
	return []outline.Option{outline.WithTimeout(seam.timeout)}
}

// kiwiSeamOptions returns the gateway options the Kiwi seam adds. Without a seam timeout it returns none.
func kiwiSeamOptions(seam seamConfig) []kiwi.Option {
	if seam.timeout <= 0 {
		return nil
	}
	return []kiwi.Option{kiwi.WithTimeout(seam.timeout)}
}

// jiraSeamOptions returns the gateway options the Jira seam adds. Without a seam timeout it returns none.
func jiraSeamOptions(seam seamConfig) []jira.Option {
	if seam.timeout <= 0 {
		return nil
	}
	return []jira.Option{jira.WithTimeout(seam.timeout)}
}

// sourceWiring is what the CLI takes from the composition root for the source commands, the protocol servers included.
type sourceWiring struct {
	service sources.Service
	serve   cli.ServeFunc
	mcp     cli.MCPFunc
}

// readSeam reads the test seam of one kind through getenv. An invalid seam is an error, never a fallback to a real
// host. Each kind reads only its own variables.
func readSeam(kind domain.SourceKind, getenv func(string) string) (seamConfig, error) {
	switch kind {
	case domain.KindOutline:
		seam, err := outline.TestSeamFromEnvWithFiles(getenv, os.ReadFile)
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
		seam, err := jira.TestSeamFromEnvWithFiles(getenv, os.ReadFile)
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

// registeredKinds returns the kinds this composition root registers, which a catalog may name. The root registers the
// three built-in kinds and no other. A kind that another build registers is passed to wireSources the same way, as a
// value, so no package holds a registry.
func registeredKinds() domain.KindSet {
	return domain.BuiltinKinds()
}

// wireSources loads the sources catalog, builds the gateway registry, and returns the source service and the serve and
// MCP functions for the configuration paths the caller resolved. The catalog is read once here. A catalog error stops the
// sources commands, and the process still starts for every other command.
func wireSources(paths configpaths.Paths, seams seamSet, h host, kinds domain.KindSet) (sourceWiring, error) {
	if err := checkSeams(seams); err != nil {
		return sourceWiring{}, err
	}
	creds, err := h.openEnvFile(paths.EnvFile)
	if err != nil {
		return sourceWiring{}, err
	}
	loader, err := h.openCatalog(paths.Catalog, kinds)
	if err != nil {
		return sourceWiring{}, err
	}
	cat, err := loader.Load()
	if err != nil {
		return sourceWiring{}, err
	}
	resolver := registry.New(map[domain.SourceKind]registry.Builder{
		domain.KindOutline: func(src domain.Source) (ports.SourceGateway, error) {
			return buildOutline(src, creds, seams[domain.KindOutline])
		},
		domain.KindKiwi: func(src domain.Source) (ports.SourceGateway, error) {
			return buildKiwi(src, creds, seams[domain.KindKiwi])
		},
		domain.KindJira: func(src domain.Source) (ports.SourceGateway, error) {
			return buildJira(src, creds, seams[domain.KindJira])
		},
	})
	svc := sources.Service{Catalog: cat, Gateways: resolver}
	// The GraphQL server serves the reads of the catalog only: its schema has no mutation.
	return sourceWiring{
		service: svc,
		serve:   graphqlServe(svc, h.stderr, graphql.Listen),
		mcp:     mcpServe(svc, h.stderr),
	}, nil
}

// buildOutline returns the gateway of one Outline source. The source's endpoint names the host that receives its key:
// the catalog is the reviewed control. The seam, when it is on, replaces only the connection base.
func buildOutline(src domain.Source, creds ports.Credentials, seam seamConfig) (ports.SourceGateway, error) {
	if !seam.on {
		return outline.New(src, creds), nil
	}
	gw, err := outline.NewWithBase(src, seam.base, seam.client, creds, seamOptions(seam)...)
	if err != nil {
		return nil, err
	}
	return gw, nil
}

// buildKiwi returns the gateway of one Kiwi source, under the same rule as buildOutline: the
// source's endpoint receives its key, and the seam, when it is on, replaces only the connection base.
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

// buildJira returns the gateway of one Jira source, under the same rule as buildOutline and buildKiwi.
// The gateway reads the token and the account e-mail address through secrets, which the credentials adapter provides
// both of.
func buildJira(src domain.Source, secrets jira.Secrets, seam seamConfig) (ports.SourceGateway, error) {
	if !seam.on {
		return jira.New(src, secrets), nil
	}
	gw, err := jira.NewWithBase(src, seam.base, seam.client, secrets, jiraSeamOptions(seam)...)
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
