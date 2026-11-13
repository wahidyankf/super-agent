# cmd/retriever

The `retriever` binary and its composition root. The package reads the configuration paths, wires the real adapters to
the core use cases, and runs the command-line interface. The core and the adapters do not import this package.

- `main.go` is the process entry point. It exits with the status that `run` returns, and it attaches the source commands
  to the command graph through `attachSources`.
- `wiring.go` builds the source gateways for Outline, Kiwi, and Jira, the catalog, and the `serve` and `mcp` servers. It
  passes the test-only seam variables to each adapter. Each adapter's seam accepts a loopback address only.
- `wiring_test.go` tests the wiring of the sources and the composition root.
- `wiring_closures_test.go` tests the closures in `wiring.go`: the seam readers, the source builders, the `serve` and
  `mcp` runners, and the `main` process path.
