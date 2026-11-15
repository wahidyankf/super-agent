# Changelog

All notable changes to super-agent are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

super-agent is alpha. Breaking changes are allowed in any release, including minor and patch releases. Each breaking
change is listed under its release with a **Breaking** label, so you can pin a version and read what changed before you
upgrade.

## Unreleased

### Breaking

- **Breaking**: the source kinds are no longer a closed list in the contract. `retriever-contract` is at `0.4.0`, and
  `catalog.NewWithFiles` takes the registered kinds as a third argument. A Go caller passes `domain.BuiltinKinds()` to
  keep the built-in set. The contract adds `KindSet`, `KindDefinition`, and `NewKindSet`, and `operations` adds
  `NewCatalog` and `BuiltinCatalog`.
- The catalog's unknown-kind message lists the registered kinds in alphabetical order. For the built-in set it reads
  `unknown kind (known: jira, kiwi, outline)` in place of `unknown kind (known: outline, kiwi, jira)`. The stable code
  `retriever.sources.unknown_source_kind` does not change, and the `--kind` flag and GraphQL messages keep their text.
- **Breaking**: the access matrix of a connector spec is strict. An `admin` operation with `GET` is now refused at load,
  where it was accepted before. A read takes `GET` or `HEAD`, or `POST` with `read_over_post: true`, and a write or an
  admin takes `POST`, `PUT`, `PATCH`, or `DELETE`. The bundled example uses none of the refused pairs.

### Added

- The Outline read tools have a golden of their MCP surface, which pins each tool's name, description, input schema, and
  hints. The reads still run on the Go gateway. The connector spec language cannot yet express them, because Outline
  sends every call as a POST, and the [connector spec fields](docs/reference/connector-spec-fields.md) page lists the
  limits.
- `libs/retriever-spec` is a Go library for the declarative connector spec of ADR 001. It reads one YAML document of at
  most 256 KiB, checks it against a JSON Schema, and refuses it with the stable code `retriever.connector.spec_invalid`
  and the path of each broken field. A valid spec gives a plain-text review of what it can do. The `retriever` command
  loads specs, as the entry below describes.
- `libs/retriever-engine` is the generic engine of ADR 001, which runs the read and write operations of a validated spec
  against one source. It renders each request from the spec, sends the key only in its auth header, and sends it over
  HTTPS with the safeguards of the ADR: no redirect is followed, a reply is capped at 10 MiB, a request at 30 seconds,
  an IP literal and a private address are never dialed, paths and headers are checked before any request, and nested
  JSON is capped. A write is sent once. A write whose outcome is unknown returns `write_outcome_unknown` with exit 2 and
  is never retried. A name that a write declares is resolved by one read, and a name with no single match sends no write
  and returns `unresolved_name`. Markdown converters are not served yet. The replay stub in `engine/replay` answers the
  read and write examples of a spec, so tests run without a network. The `retriever` command runs it for each loaded
  spec, as the entry below describes.
- `retriever` loads connector specs. A folder of spec files, named by `--connectors-dir` or else the default
  `retriever/connectors` folder of the user configuration folder, adds each spec's kind to the catalog. Each operation
  of a spec with `cli` on becomes a command under a group named by the kind, such as `example-wiki docs list`, and each
  operation with `mcp` on becomes an MCP tool. A catalog source may pin a spec with `spec_sha256`, and a mismatch stops
  the run before any request goes out. A refused spec or pin exits `2` with `retriever.connector.spec_invalid` or
  `retriever.connector.spec_pin_mismatch`, and an explicit folder that is missing or unreadable exits `2` with
  `retriever.connector.folder_unavailable`. The bundled example is `apps/retriever-cli/examples/connectors/example-wiki.yml`,
  and the [add a spec connector](docs/how-to/add-a-spec-connector.md) guide explains the steps. A spec operation that
  names a Markdown converter loads, and each run of it fails with `retriever.<kind>.unsupported_operation` before any
  request. In this release the `--kind` filter of `sources list` and GraphQL accept the built-in kinds only, and a spec
  kind's sources appear in an unfiltered `sources list`. The `connector validate` and `connector test` commands are not
  built yet.
- A connector spec read may be sent as `POST`, with `access: read`, `method: POST`, and `read_over_post: true` (ADR 002,
  E1). Its body is a JSON object of constants and declared body parameters, sent with `Content-Type: application/json`.
  A spec may not set `Content-Type` or `resolve` on such a read. The body is capped at 64 KiB and at 8 levels of
  nesting, and a body over either cap fails with `retriever.<kind>.body_limit` before any request. The review text lists
  each POST read. The bundled example adds `example-wiki docs find`, a read over POST.
- An MCP tool of a POST read reports the hints of a write, so no client runs it as read-only, until every source of its
  kind pins the spec with `spec_sha256`. Then it reports read hints. The command line and the routing do not change.
- `retriever-contract` adds the field `Row.WriteHints`, and `retriever-runtime` adds `Loader.PinnedKinds()`, which the
  `retriever` command uses for the hint rule.
- A connector spec read may page in the JSON body, with `pagination.in: body` (ADR 002, E2). With `in: body`, every
  parameter the paging rule names must be a declared body parameter of the same operation, and a rule cannot mix
  locations. The offset is a non-negative integer below 2^31. The engine advances it by the number of items in the
  page, stops at the first short page, and at the cap returns the pages it has with a continuation; a next cursor that
  repeats one already sent ends the read the same way. The Go `collections.list` keeps failing at the cap with a
  page-cap error, while the engine returns pages plus a continuation. The review text lists each body-paged read.
- A reply field of a connector spec takes `required` and a closed `format` (ADR 002, E3). A missing, null, or empty
  value for a field with `required: true` fails the read with `retriever.<kind>.unexpected_response`, and no partial
  record is returned. A field with `format` is checked against one closed name, `id`, `uuid`, or `share-id`, with the
  same checks the catalog kinds use on request parameters; a free pattern has no form in the language. The failure
  names the field and never the value. The review text lists each operation with required or checked fields. A request
  with a missing argument still sends no request.

### Changed

- The repository page shows the root `README.md`. The `.github/README.md` file is removed, because GitHub shows that
  file in place of the root one. Its content moved to the [repository layout
  reference](docs/reference/repository-layout.md).
- The retriever code moves out of the app into six Go modules under `libs/`: `retriever-contract`, `retriever-runtime`,
  `retriever-testkit`, `retriever-connector-outline`, `retriever-connector-kiwi`, and `retriever-connector-jira`. The
  libraries are at `v0.4.0`, and the contract's `ContractVersion` is `0.4.0`. No library tag is cut yet, so an outside
  build cannot require the libraries until one is.
- The split changes no behaviour of the `retriever` command-line tool: no command, flag, exit status, or JSON output
  changes.
- `retriever-spec` and `retriever-engine` are at `v0.5.0`, and the other libraries stay at `v0.4.0`. The connector spec
  language gains POST reads, and the engine gains the `body_limit` code.
- Each Go module has its own coverage gate, held to a 99 percent floor. The app and each library are measured
  separately, and no profile merges them.

## v0.2.0

Alpha pre-release. Expect breaking changes: commands, flags, file formats, connector specs, JSON output, and APIs can
change in any release, including minor and patch releases, until a stable 1.0. Read the
[migration table](docs/reference/retriever-cli-migration-from-butler.md) before you move from Butler, and the
[scenario-gap page](docs/reference/retriever-cli-scenario-gap.md) for the behaviours that no scenario covers yet. The
breaking changes below are the ones made since `v0.1.0`.

### Breaking

- **Breaking**: the access levels are removed, with the per-kind ceilings, the `--max-access` flag, and the `access`
  column. `retriever` no longer checks what a command may change. The API token of a source decides what that source
  allows, so limit the token to the reads or writes you want. A catalog that still has an `access:` key is refused when
  it loads, with `retriever.sources.catalog_invalid`. `--max-access` is an unrecognized argument and exits `2`. The
  `access` field is gone from `retriever --output json sources list`, from the GraphQL `Source` type, and from the MCP
  `list_sources` result. The JSON sources list is now `schemaVersion` 2, because removing `access` is a breaking change.
  Every other JSON document stays at `1`.
- **Breaking**: the confirmation and preview gates are removed. `--yes`, `--confirm`, and `--dry-run` are unrecognized
  arguments, and each exits `2` with `retriever.<area>.usage_unrecognized_argument`. A destructive or admin command runs
  the change it names, so a script that passed `--yes` must drop it. A write no longer takes `--dry-run`. The error codes
  `yes_required`, `confirm_required`, `confirm_mismatch`, `self_target`, and `last_revision_required` are gone. A document
  update no longer needs `last_revision`.
- **Breaking**: a `retriever` read on Outline, Kiwi, or Jira that a rate limit stops now exits `124`, not `2`. Exit `124`
  means a bounded wait ended in a retryable failure, the same status a timeout uses, and the JSON error still says
  `retryable: true`. A `Retry-After` longer than 60 seconds still exits `2`, because that failure is not retryable.
- **Breaking**: `retriever mcp` exits `2` when a session is cancelled before any signal arrives. Before this release the
  exit status could be `0`, depending on timing. A cancelled session does not complete, so it now exits with the status
  the exit-code contract names for that case.
- **Breaking**: `retriever mcp` offers every tool of the catalog, so the tool list no longer depends on an access level.
  The server offers 165 tools, and `--tools` still limits the resources. Eleven of the tools are write and admin tools
  that had no MCP tool before: `empty_trash`, `delete_user`, `delete_all_groups`, `delete_team`,
  `delete_authentication_provider`, `create_integration`, `update_integration`, `create_oauth_client`,
  `rotate_oauth_client_secret`, `create_webhook_subscription`, and `update_webhook_subscription`. The MCP write and admin
  tools no longer take a `dry_run` input.
- **Breaking**: `retriever outline export`, `kiwi export`, and `jira export` name the file by the document ID, `<id>.md`.
  Outline exports were named by the title slug, and Kiwi and Jira exports by the ID and a slug. A file of the same name is
  replaced without a collision check. A script that looked for a file by its title must look for `<id>.md` instead.

### Changed

- The behaviour specs grow from 156 scenarios in 13 feature files at `v0.1.0` to 333 scenarios in 14 feature files. The
  new scenarios cover the transport limits (HTTPS, redirects, timeouts, rate-limit waits, and reply size), the
  credentials and secret redaction, the catalog and env-file rules, the GraphQL request guards, the export file names,
  and the Outline, Jira, and Kiwi read contracts.
- `retriever` unit tests no longer read or write real files, read or change the process environment, change the working
  directory, or start child processes. The tests that need those move to files with the `integration` build tag. The
  behaviour harness runs its `@unit` scenarios in memory. This changes the tests, not the behaviour of any command.
- The architecture guard that keeps untagged tests hermetic has no exemption. It flags a reference to the disk, the
  environment, the working directory, a child process, or a pipe, even when the call sits in a helper.
- The unit coverage floor stays at 99 percent, and no exclusion is added.

### Fixed

- The public-safety scanner no longer reports a Go package-qualified identifier as an internal hostname. The release
  binaries name the standard library's `crypto/x509` type `local` in that form, so the `internal-hostname` rule blocked
  every binary although it names no host. The rule now skips only that shape. URLs, absolute paths, dotted or hyphenated
  paths, and bare host names still block. The release procedure now screens the unpacked binaries.

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
