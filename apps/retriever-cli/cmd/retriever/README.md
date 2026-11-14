# cmd/retriever

The `retriever` binary and its composition root. The package holds the real wrappers that reach the process: the
standard streams, the environment, the working directory, the clock, and the disk. It wires those wrappers and the
connectors to the libraries, and it runs the command-line interface. The libraries and the adapters do not import this
package.

- `main.go` is the process entry point. It exits with the status that `run` returns. It attaches the source commands
  through `attachSourcesFrom`, which loads the connector specs first, so their commands exist before the catalog is read.
- `host.go` defines the host that `run` takes: the streams, the working directory, the clock, the export sink, the
  connectors folder opener, and the engine ports. `realHost` is the host of the real process, and only `main` passes it.
- `disk.go` holds the real openers. They pass `os.DirFS`, `os.CreateTemp`, and the other `os` calls to the libraries as
  functions, so the libraries receive those calls and do not reach the disk on their own.
- `connectors.go` reads the connectors folder and validates each spec. A folder that `--connectors-dir` names must
  exist, and a missing default folder loads no spec. It also builds one engine for each spec kind.
- `specnet.go` holds the real clock and the real network of the engine: the system resolver and the standard dialer.
- `wiring.go` builds the registry of source builders for Outline, Kiwi, Jira, and each spec kind, the catalog, and the
  `serve` and `mcp` servers. It passes each connector's test-only seam, which accepts a loopback address only.

The tests here cover the wiring, the host, the openers, and the spec folder. `wiring_closures_integration_test.go`,
`wiring_files_integration_test.go`, and `connectors_integration_test.go` carry the `integration` build tag. The last one
runs the bundled example spec over loopback HTTPS against the stub in `tests/specstub`. `disk_open_test.go` calls the
openers on absolute paths; building a loader reads no file. The other tests use a fake host.
