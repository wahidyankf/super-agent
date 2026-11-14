# internal/adapters/in/cli

The command-line interface. It parses the arguments, calls the use cases through the ports it is given, and writes the
text or JSON output and the failure body. The exit statuses are in
[exit codes](../../../../../../docs/reference/retriever-cli-exit-codes.md).

Production files:

- `cli.go` holds the root command, the global options, and the output writers.
- `command.go` holds the command type, the help text, and the exit-status table that help prints.
- `commands.go` builds the command tree and sets the signal and exit rows of each command.
- `catalog_paths.go` passes the catalog and env-file flags to the configuration paths.
- `environment.go` builds the lookup of the source keys that the composition root passes in as `getenv`. It reads only
  the variables that the sources name.
- `failure.go` writes the failure body, in text or JSON.
- `sources.go` holds the `sources` and `document` commands.
- `outline.go`, `outline_read.go`, and `outline_write.go` hold the `outline` commands, split by read and write.
- `serve.go` holds the `serve` command, and `mcp.go` holds the `mcp` command.
- `signals.go` gives the cancellation cause that ends `serve` or `mcp` as if the process had received a signal.
- `pipe.go` maps a closed standard output or standard error to the status of a process that ends by `SIGPIPE`.

Tests sit beside the code. `environment_test.go` reads the real environment, so it carries the `integration` build tag.
The untagged tests use the fake services in `outline_fakes_test.go` and `service_fakes_test.go`.
