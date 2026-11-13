# internal/adapters/in/cli

The command-line interface. It parses the arguments, calls the core use cases through the ports it is given, and writes
the text or JSON output and the failure body. The exit statuses are in
[exit codes](../../../../../../docs/reference/retriever-cli-exit-codes.md).

Source files:

- `cli.go` holds the root command, the global options, and the output writers.
- `command.go` holds the command type, the help text, and the exit-status table that help prints.
- `commands.go` builds the command tree and sets the signal and exit rows of each command.
- `catalog_paths.go` passes the catalog and env-file flags to the configuration paths.
- `environment.go` builds the process environment that the commands read.
- `failure.go` writes the failure body, in text or JSON.
- `sources.go` holds the `sources` and `document` commands.
- `outline.go`, `outline_read.go`, and `outline_write.go` hold the `outline` commands, split by read and write.
- `serve.go` holds the `serve` command.
- `mcp.go` holds the `mcp` command.

Test files:

- `cli_test.go` tests the root command, the global options, and the output modes.
- `help_exit_test.go` checks that each help page lists the exit statuses that the command can return.
- `public_errors_test.go` checks that the public error texts do not echo a value.
- `environment_test.go` tests the environment builder. It carries the `integration` build tag.
- `environment_unit_test.go` tests the same builder without the tag.
- `sources_test.go`, `outline_test.go`, `mcp_test.go`, and `serve_test.go` test each command group.
- `kind_commands_test.go`, `kind_error_test.go`, and `identity_error_test.go` test the kind rules and their errors.
- `outline_errors_test.go` and `outline_messages_test.go` test the outline errors and their messages.
- `outline_read_keys_test.go`, `outline_read_page_test.go`, `outline_read_terminal_test.go`, and
  `outline_read_usage_test.go` test the outline read commands: keys, paging, terminal output, and usage.
- `outline_fakes_test.go` and `service_fakes_test.go` hold the fake services that the tests use.
- `mcp_signal_help_test.go` checks that `mcp --help` lists the exit statuses for an interrupt and for `SIGTERM`.
- `serve_signal_integration_test.go` sends a real signal to `serve`. It carries the `integration` build tag.
