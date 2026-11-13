// Command retriever is the composition root: it resolves the configuration paths, wires the real adapters to the core
// ports, and hands control to the CLI adapter.
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/configpaths"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/in/cli"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/adapters/out/markdownfiles"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/ports"
)

func main() {
	os.Exit(run())
}

// run builds the dependencies, runs one command line, and returns the process exit code.
func run() int {
	argv := os.Args[1:]

	// An invalid test seam of any kind stops every command. It must never fall back to a production host.
	seams, err := readSeams(os.Getenv)
	if err != nil {
		return startupFailure(os.Stderr, argv, err)
	}

	// A malformed command line is reported by cli.Run, which parses it again, so the path options are read best-effort here.
	globals, _ := cli.ParseGlobals(argv)

	deps := cli.Deps{
		Ports:  ports.Ports{},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getenv: cli.SecretEnv,
		Now:    time.Now,
		Sink:   exportSink,
	}
	deps.SetupErr = attachSources(&deps, globals, seams)
	return cli.Run(argv, deps)
}

// exportSink returns the sink that writes the exported Markdown files into the folder the --out flag names.
func exportSink(dir string) ports.ExportSink {
	return markdownfiles.New(dir)
}

// startupFailure reports a failure that stops the process before the CLI runs, and returns its exit status. The message
// goes to stderr only. Every failure of the tool that is not a result exits 2 (CLI convention 001).
func startupFailure(stderr io.Writer, argv []string, err error) int {
	fmt.Fprintf(stderr, "retriever: %v\n", err)
	return cli.StartupStatus(argv)
}

// attachSources resolves the configuration paths and wires the sources of every kind into deps. A failure is returned, so
// the CLI records it as the setup error, and only the commands that need the sources report it.
func attachSources(deps *cli.Deps, g cli.Globals, seams seamSet) error {
	return attachSourcesFrom(deps, g, seams, os.Getenv, os.Getwd)
}

// attachSourcesFrom is attachSources with the environment and the working directory passed in, so a test can make the
// path rules fail without changing the real process.
func attachSourcesFrom(deps *cli.Deps, g cli.Globals, seams seamSet, getenv func(string) string, getwd func() (string, error)) error {
	paths, err := configpaths.Resolve(configpaths.Inputs{
		CatalogFlag: g.Catalog,
		EnvFileFlag: g.EnvFile,
		Getenv:      getenv,
		Getwd:       getwd,
		Exists:      regularFileExists,
	})
	if err != nil {
		return err
	}
	w, err := wireSources(paths, seams)
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
