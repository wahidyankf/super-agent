// Command retriever is the composition root: it resolves the configuration paths, wires the real adapters to the core
// ports, and hands control to the CLI adapter.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/libs/retriever-contract/ports"
	"github.com/wahidyankf/super-agent/libs/retriever-runtime/configpaths"
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, realHost))
}

// run builds the dependencies, runs one command line, and returns the process exit code. The command line, the
// environment, and the host are parameters, so a test runs any command line with fakes and changes no real process state.
func run(argv []string, getenv func(string) string, h host) int {
	// An invalid test seam of any kind stops every command. It must never fall back to a production host.
	seams, err := readSeams(getenv)
	if err != nil {
		return startupFailure(h.stderr, argv, err)
	}

	// A malformed command line is reported by cli.Run, which parses it again, so the path options are read best-effort here.
	globals, _ := cli.ParseGlobals(argv)

	deps := cli.Deps{
		Ports:  ports.Ports{},
		Stdin:  h.stdin,
		Stdout: h.stdout,
		Stderr: h.stderr,
		Getenv: cli.SecretLookup(getenv),
		Now:    h.now,
		Sink:   h.exportSink,
	}
	deps.SetupErr = attachSourcesFrom(&deps, globals, seams, getenv, h)
	return cli.Run(argv, deps)
}

// exportSink returns the sink that writes the exported Markdown files into the folder the --out flag names.
func exportSink(dir string) ports.ExportSink {
	return exportFolder(dir)
}

// startupFailure reports a failure that stops the process before the CLI runs, and returns its exit status. The message
// goes to stderr only. Every failure of the tool that is not a result exits 2 (CLI convention 001).
func startupFailure(stderr io.Writer, argv []string, err error) int {
	fmt.Fprintf(stderr, "retriever: %v\n", err)
	return cli.StartupStatus(argv)
}

// attachSourcesFrom resolves the configuration paths and wires the sources of every kind into deps. The environment and
// the host are passed in, so a test can set the path rules and the files without changing the real process. A failure is
// returned, so the CLI records it as the setup error, and only the commands that need the sources report it.
func attachSourcesFrom(deps *cli.Deps, g cli.Globals, seams seamSet, getenv func(string) string, h host) error {
	paths, err := configpaths.Resolve(configpaths.Inputs{
		CatalogFlag: g.Catalog,
		EnvFileFlag: g.EnvFile,
		Getenv:      getenv,
		Getwd:       h.getwd,
		Exists:      h.exists,
	})
	if err != nil {
		return err
	}
	w, err := wireSources(paths, seams, h)
	if err != nil {
		return err
	}
	deps.Service, deps.Serve, deps.MCP = w.service, w.serve, w.mcp
	return nil
}

// regularFileExists reports whether a regular file exists at path.
func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
