# cmd/retriever

The `retriever` binary and its composition root. The package holds the real wrappers that reach the process: the
standard streams, the environment, the working directory, the clock, and the disk. It wires those wrappers and the
connectors to the libraries, and it runs the command-line interface. The libraries and the adapters do not import this
package.

- `main.go` is the process entry point. It exits with the status that `run` returns, and it attaches the source commands
  through `attachSourcesFrom`.
- `host.go` defines the host that `run` takes: the streams, the working directory, the clock, and the export sink.
  `realHost` is the host of the real process, and only `main` passes it.
- `disk.go` holds the real openers. They pass `os.DirFS`, `os.CreateTemp`, and the other `os` calls to the libraries as
  functions, so the libraries receive those calls and do not reach the disk on their own.
- `wiring.go` builds the registry of source builders for Outline, Kiwi, and Jira, the catalog, and the `serve` and `mcp`
  servers. It passes each connector's test-only seam, which accepts a loopback address only.

The tests here cover the wiring, the host, and the openers. `wiring_closures_integration_test.go` and
`wiring_files_integration_test.go` carry the `integration` build tag. `disk_open_test.go` calls the openers on absolute
paths; building a loader reads no file. The other tests use a fake host.
