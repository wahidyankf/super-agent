# Changelog

All notable changes to super-agent are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

super-agent is alpha. Breaking changes are allowed in any release, including minor and patch releases. Each breaking
change is listed under its release with a **Breaking** label, so you can pin a version and read what changed before you
upgrade.

## Unreleased

### Breaking

- **Breaking**: a `retriever` read on Outline, Kiwi, or Jira that a rate limit stops now exits `124`, not `2`. Exit
  `124` means a bounded wait ended in a retryable failure, the same status a timeout uses, and the JSON error still
  says `retryable: true`. A `Retry-After` longer than 60 seconds still exits `2`, because that failure is not
  retryable.
- **Breaking**: `retriever` no longer gates the changes it runs against a source. The confirmation and preview gates are
  removed. `--yes`, `--confirm`, and `--dry-run` are unrecognized arguments now, and each exits `2` with
  `retriever.<area>.usage_unrecognized_argument`. A destructive or admin command runs the change it names, so a script
  that passed `--yes` must drop it. The error codes `yes_required`, `confirm_required`, `confirm_mismatch`,
  `self_target`, and `last_revision_required` are gone. A document update no longer needs `last_revision`.
- **Breaking**: the MCP server gains eleven write and admin tools that had no tool before: `empty_trash`, `delete_user`,
  `delete_all_groups`, `delete_team`, `delete_authentication_provider`, `create_integration`, `update_integration`,
  `create_oauth_client`, `rotate_oauth_client_secret`, `create_webhook_subscription`, and
  `update_webhook_subscription`. The MCP write and admin tools no longer take a `dry_run` input.
- **Breaking**: the access levels are removed, with the per-kind ceilings and the `--max-access` flag. `retriever` no
  longer checks what a command may change. The API token of a source decides what that source allows, so limit the
  token to the reads or writes you want. A catalog that still has an `access:` key is refused when it loads, with
  `retriever.sources.catalog_invalid`. `--max-access` is an unrecognized argument and exits `2`. The `access` field is
  gone from `retriever --output json sources list`, from the GraphQL `Source` type, and from the MCP `list_sources`
  result. The JSON sources list keeps `schemaVersion` 1.
- **Breaking**: `retriever mcp` offers every tool of the catalog. `--tools` still limits the resources, and the tool
  list no longer depends on an access level.

### Changed

- The behaviour specs add 26 scenarios, from 194 to 220, for the transport limits (HTTPS, redirects, timeouts,
  rate-limit waits and spacing, and reply size), the Kiwi and Jira error codes, the Jira search and projects reads, and
  the redaction of MCP replies. The behaviour harness now drives the in-process Outline, Kiwi, and Jira stubs with a
  fake clock, so the waits are checked without real sleeping at the unit and integration levels.
- `retriever` unit tests no longer read or write real files, read or change the process environment, change the working
  directory, or start child processes. The tests that need those move to files with the `integration` build tag. The
  unit tests reach the file system and the environment through fakes at the seams of the catalog, the credentials, the
  Markdown sink, the source seams, and the composition root. The behaviour harness runs its 220 `@unit` scenarios in
  memory, through an in-memory file tree and an in-memory export sink. The integration and end-to-end levels keep the
  real disk, the real environment at the integration level, and the built binary at the end-to-end level. The synthetic
  Outline replies move to `tests/support/synthetic`.
- The architecture guard that keeps untagged tests hermetic has no exemption. It reads the source of every untagged test
  file, and it flags a reference to the disk, the environment, the working directory, a child process, or a pipe, even
  when the call sits in a helper. Its negative tests run in `test:unit`.
- The unit coverage floor stays at 99 percent, and no exclusion is added. The real-disk wrappers of the catalog, the
  credentials, the Markdown sink, and the composition root now run only at the integration and end-to-end levels, so the
  unit figure moves to 99.01 percent.

### Fixed

- The public-safety scanner no longer reports a Go package-qualified identifier as an internal hostname. The release
  binaries name the standard library's `crypto/x509` type `local` in that form, so the `internal-hostname` rule blocked
  every binary although it names no host. The rule now skips only that shape. URLs, absolute paths, dotted or hyphenated
  paths, and bare host names still block. The release procedure now screens the unpacked binaries.
- `retriever mcp` no longer exits `0` when a session is cancelled before any signal arrives. A race between the end of
  the session and its cancellation made the exit status depend on scheduling. A cancelled session now exits `2`, as the
  exit-code contract says.

## v0.1.0

Alpha pre-release. Expect breaking changes: commands, flags, file formats, connector specs, JSON output, and APIs can
change in any release, including minor and patch releases, until a stable 1.0. Read the
[migration table](docs/reference/retriever-cli-migration-from-butler.md) before you move from Butler, and the
[scenario-gap page](docs/reference/retriever-cli-scenario-gap.md) for the behaviours that no scenario covers yet. This is
the first release, so it has no earlier version to break.

### Added

- Repository skeleton: Nx workspace configuration, HIPPO and RHINO pins, Husky hooks, public-safety gates, and the
  governance and docs trees.
- `retriever`, the sourcing command-line tool in `apps/retriever-cli`. It lists the sources of a catalog, reads
  documents
  from Outline, Kiwi, and Jira, writes Markdown exports to a folder, and serves a read-only GraphQL API (`serve`) and a
  Model Context Protocol server (`mcp`). It is alpha. See the
  [command reference](docs/reference/retriever-cli-commands.md).
- Documentation for `retriever`: a how-to for a first source, and references for the commands, exit codes, JSON
  schemas, configuration, and catalog format. The
  [migration table](docs/reference/retriever-cli-migration-from-butler.md)
  maps each behaviour of the older Butler tool to its `retriever` counterpart.
  - `serve` and `mcp` now share one signal rule: `SIGINT` exits `130` and `SIGTERM` exits `143`, and a closed standard
    output pipe ends the process by `SIGPIPE` (shell status `141`).
  - The migration table now lists four deliberate departures from Butler: refusals before reads, the missing-catalog
    code, the HTTPS refusal at load, and the unknown outcome of a Kiwi write. `retriever` supports macOS and Linux only.
- A release build, `build:release`, that writes one archive for each of `darwin` and `linux` on `amd64` and `arm64`, and
  a `SHA256SUMS` file, from an annotated tag.
