# Changelog

All notable changes to super-agent are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

super-agent is alpha. Breaking changes are allowed in any release, including minor and patch releases. Each breaking
change is listed under its release with a **Breaking** label, so you can pin a version and read what changed before you
upgrade.

## Unreleased

### Added

- `retriever-conformance` checks any connector spec: the schema is valid, every operation carries examples, each
  example replays through the engine and matches its expected fields, and the CLI, MCP, and README texts derived from
  the spec equal the stored texts. `retriever connector validate <spec>` runs the static half and prints the review,
  and `retriever connector test <spec>` runs the whole suite and prints the report. A passing run exits `0` with the
  report on standard output; a failing run exits `2` with redacted diagnostics on standard error and nothing on
  standard output. The suite runs over every bundled spec present (`outline.yml` and `jira.yml`; the Kiwi scenarios
  stay Go-served and out of scope), checks every `catalog_id`, and refuses a bundled host outside `example.test`,
  `example.invalid`, and `example.com`. A canonical UUID in an example is an ID, not a token-shaped run: the R2
  example guard exempts it, so UUID operations carry replayable examples, and the bundled `outline.yml` examples use
  UUIDs where a UUID is required. The bundled specs gain no new operations.
- The bundled `jira.yml` serves the five Jira catalog reads, each with a `catalog_id` and an example. The Jira
  gateway runs every operation on the bundled-spec engine, with the document-not-found translation the Go
  gateway maps. `whoami` ports with no new feature; the collection walk is a second offset read of the project
  search with the page-end rule, and the tree walk is a second cursor read of the search endpoint with the page-end
  rule, reusing the `IssueKey` check, the text converters, the key echo, and the built IDs. The unchanged Jira
  read, redaction, and transport scenarios run on the engine: retries and pages wait the 100 ms pacing of the Go
  gateway, and the project list and the comment list send the Jira wire names through the `query_key` alias, `limit`
  as `maxResults` and `offset` as `startAt`. The document layout names flat output-field names only, and the
  intermediate fields stay in the record beside the five catalog fields of `issues.get`. The spec measures
  7 operations in 14,946 bytes, 6 percent of the 256 KiB cap, and `TestBundledSpecCoversEveryCatalogRow` covers the
  Jira kind with no scenario edited.
- The Jira gateway runs every Jira operation on the bundled-spec engine, with the document-not-found translation
  the retired Go gateway mapped. `retriever-connector-jira` leaves the dependency graph: the app no longer imports
  it, and the architecture test fails on any new import. The Jira seam and the issue and project ID checks move
  into the app at `apps/retriever-cli/internal/jiraseam` with the environment variable names unchanged. The
  unchanged Jira read, redaction, and transport scenarios pass on the engine with no edits.

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

- The spec language admits the `query_key` wire alias on `in: query` parameters: the engine sends the query pair
  under the alias while the parameter keeps its catalog name everywhere else (the CLI flag, the examples, and the
  review). The alias starts with a letter and holds letters, digits, and underscores only, covering camelCase wire
  names such as `maxResults` and `startAt`; an alias on a parameter outside the query, one repeating the parameter
  name, and one colliding with another query key of the same operation are refused at load. The schema stays at
  version 1; `retriever-spec` and `retriever-engine` stay at `v0.6.0`.
- The spec language admits the document layout on a read of one object: the title renders as `# <title>`, each line
  renders `- <label>: <value>` in one closed kind (`field` escapes its value with the moved `escapeText` algorithm,
  `verbatim` writes a shape-checked key or date as is, and `pair` renders `name (key)`), and the body renders
  `## <heading>` with the converted description, or the empty constant when it is blank. `join` renders a string
  list with the fixed separator `", "`, `default` fills an empty value, `skip_empty` omits the line when the value
  is still empty, and `valid` empties a value outside the issue-key shape. Planted markup stays literal text. The
  schema stays at version 1; `retriever-spec` and `retriever-engine` stay at `v0.6.0`.
- The spec language admits the key echo on a read of one object: after mapping, the text at the echo pointer must
  equal the request argument exactly, or the read fails with `unexpected_response` naming the pointer path, never the
  values. A reply field may build its value from an issue key instead of reading it: `project_key` (the text before
  the last hyphen), `issue_id` (the key prefixed with `issue-`), `collection_id` (the project part prefixed with
  `project-`), and `browse_url` (the endpoint origin joined with `/browse/` and the key, never a URL the reply
  carries). All four share one closed issue-key check, and a value outside the shape fails with `unexpected_response`
  naming the field, never the value. The schema stays at version 1; `retriever-spec` and `retriever-engine` stay at
  `v0.6.0`.
- The spec language admits the `jira-time`, `jira-date`, and `jira-title` converters as pure functions of the reply:
  a Jira timestamp answers as RFC 3339 in UTC, a calendar day passes unchanged, and a title keeps one line of at
  most 255 runes with no hidden character. Any other value, including a missing one, maps to the empty text with
  the field present, so the row is kept with a visible gap, exactly as the Go code answers today. The moved Go
  parity tests pass unchanged against the engine functions. The schema stays at version 1; `retriever-spec` and
  `retriever-engine` stay at `v0.6.0`.
- The spec language admits the `IssueKey` path parameter kind with the shared strict check of the catalog: a value
  outside the shape fails before any request with `invalid_document_id`, and never echoes the value. An offset read
  may declare the page-end rule: `last_pointer` names the reply flag that decides last, beating the short-page rule,
  and `cut_to_limit` names the `Limit` or `Int` parameter whose value cuts an over-limit page to its first items. A
  cursor rule declaring `last_pointer` may declare the computed `truncated` block, so each item tells whether a
  next-page token is present while the reply is not last. The schema stays at version 1; `retriever-spec` and
  `retriever-engine` are at `v0.6.0`.
- The engine serves the `markdown-from-adf` converter as a pure function of the reply, so an operation that names it
  no longer fails. The Jira converter tests pass unchanged against the moved engine function, and a document tree
  nested past the depth cap ends with the truncation marker. The Jira module keeps its copy of the function until it
  retires.
- The spec language admits the `Icon`, `URLList`, and `EventList` parameter kinds with exactly the catalog
  validation, and the bundled `outline.yml` claims the last 7 unclaimed write rows: the template, group, OAuth
  client, and webhook subscription creates and updates. Every catalog row is claimed and the unclaimed set is empty.
  The spec measures 166 operations in 244 KiB of the 256 KiB cap.
- The spec language pages an operation with no page parameters through the `page-all` rule: the fixed page size and
  the page cap, with the paging keys sent in the query or the JSON body. `collections.list` pages with it and fails
  at the cap with the page-cap error, mirroring the Go reader. The engine takes a timeout option so the test seam
  bounds an engine request the same way it bounds a Go gateway request.
- A reply field maps two reply shapes through `fallback_pointer`, and a response tells a missing object from a
  missing method through `unavailable_message`: a 404 carrying exactly the declared body message fails as
  `operation_unavailable`, matched by equality only. `documents.get` maps its wrapped and unwrapped shapes and keeps
  both 404 meanings, and now runs on the engine.
- The spec language reads editor data as plain text through `convert: text-from-editor-data`, a pure function of the
  reply. The comment reads map it, falling back to the raw text when a reply carries no editor data.
- A reply field holds one open-key map with closed bool values through `bool_map: true`: the engine keeps every key
  the reply holds, verbatim, sorted, with each value checked as a flag. The two notification subscription writes run
  on the engine with it, so all 99 Outline writes run on the engine. A value that is not a bool fails the read with
  `unexpected_response` naming the dotted path instead of being dropped, which is stricter than the retired Go
  projection and accepted. Any other open key set stays refused: a star sub-field, `patternProperties`,
  `additionalProperties`, or an undeclared value rule is refused at load.
- The bundled `outline.yml` claims all 99 Outline write, destructive, and admin rows, each with a `catalog_id`
  and an example. The Outline gateway runs every claimed operation on the bundled-spec engine, with the not-found
  translation the retired Go gateway mapped. `retriever-connector-outline` leaves the dependency graph: the app no
  longer imports it, and the architecture test fails on any new import. The unchanged write and
  destructive scenarios pass on the engine with no edits, and the MCP surface of the 65 read tools stays
  byte-identical to its golden.
- All 66 Outline reads run on the bundled-spec engine, including the comment reads, the key, authentication,
  integration, webhook, OAuth client, and sign-in provider reads, with `collections.list` paging under the
  `page-all` rule and `documents.get` mapping its wrapped and unwrapped shapes through fallback pointers. The
  unchanged read, redaction, and transport scenarios pass on the engine.
- The spec language maps a list of strings with `format: strings` and the host of an address with
  `convert: hostname`, so the engine serves reply lists and address hosts without new keywords. Every previously valid
  spec still validates, and the schema keyword lock is unchanged.
- The engine merges each page of a merged list read that pages, instead of failing it. The bundled
  `suggestions.mention` example names empty lists instead of nulls, matching the wire.
- The connector spec names catalog parameters exactly: names start lower-case and may hold upper-case letters, and the
  `CollectionID`, `CollectionUUID`, and `DocumentID` parameter kinds join the 21 existing kinds. They validate as an
  opaque ID or UUID. The bundled Outline port needs both, and every previously valid spec still validates.
- The Outline read tools have a golden of their MCP surface, which pins each tool's name, description, input schema, and
  hints.
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
  kind's sources appear in an unfiltered `sources list`. The `connector validate` and `connector test` commands are built
  below.
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
- A reply field of a connector spec takes one level of `object` with an allow-list of at most 32 sub-fields (ADR 002,
  E4). Each sub-field is a pointer with its own optional `required`, `format`, and `convert`; a sub-field with its own
  object is refused, and so is an object field with a scalar keyword of its own. A pointer that reaches an object or
  a list is refused, so a whole object is never copied. The mapper reads declared pointers only, the reply tree is
  dropped after mapping, and a planted e-mail reaches no output, error, or log. A failure names the dotted path and
  never the value. The review text lists each operation with a nested record field, with the object field and each
  sub-field. Renderers, the MCP adapter, and the JSON output already read a nested record, so the contract is
  unchanged.
- A response of a connector spec takes `first` with a JSON pointer to a list, to map the first item of the list
  (ADR 002, E5). The engine takes item 0 and maps its fields with pointers relative to the item; only index 0
  exists, and the review text says `first of /data`. An empty list ends with `retriever.<kind>.not_found` and no
  partial record, and a value that is not a list ends with `unexpected_response`. A response holds at most one of
  `items`, `first`, and `merge`. A first-item read returns one record, as a whole-reply read does, so the renderers,
  the MCP adapter, and the JSON output are unchanged.
- A response of a connector spec takes `merge` with a label field and two to eight lists, to merge the lists into
  one output (ADR 002, E6). Each list names `items`, a constant `label` of lowercase letters and hyphens of at most
  32 characters, and its own `fields`; a merge response names no top-level `fields` entry. The output keeps the
  declared list order, then the reply order, and it shares the output cap of any list. The label beats a reply field
  of the same name. A response holds at most one of `items`, `first`, and `merge`. The review text lists each merge
  read with its label field and its lists, and it joins the required, checked, and nested fields of every list under
  the same operation. A merge returns a list record, as an items read does. The command-line list rendering for a
  merge read is left to a later pull request.
- A response of a connector spec takes `tree` with `children`, `parent_field`, `depth_field`, an optional
  `root_parent_param`, and an optional `max_depth`, to flatten the nested nodes of the `items` list depth first
  (ADR 002, E7). The engine computes only the parent and the depth of each node: the parent is the id of the parent
  node, null for a top node or the value of `root_parent_param`, and the depth counts from 0. The response holds a
  required `id` field, a node deeper than the cap with its subtree is left out, and `max_depth` is at most 64. The
  computed fields beat a reply field of the same name. A reply with `tree` decodes to a fixed higher cap of 146
  levels, which covers a tree of 70 levels; no spec field sets it, and no other reply changes. A response holds at
  most one of `items`, `first`, and `merge`, and a tree rides on `items`. The review text lists each tree read with
  its pointers, its computed fields, and its depth cap. A tree returns a list record, as an items read does, and each
  page of a paged tree read flattens on its own. The collection-tree gateway role reads the computed `parent_id` and
  `depth` fields. The higher decode cap is an owner proposal that the owner can still veto at review.
- A read of one object of a connector spec takes `fallback` with the target `operation`, the one closed trigger
  `when`, and the `fill` fields (ADR 002, E8). The trigger is `empty` with a mapped output field that is empty, or
  `status` with a list of HTTP statuses. The target is a read of the same spec: it declares only parameters the
  first operation also declares and receives their values, and it has no fallback of its own, so a chain of two
  fallbacks is refused at load. At most one fallback request runs per call, and both requests share the 30 second
  limit and run under every safeguard of the engine: the same host, the same key, no redirect, and the same caps.
  When the fallback fails, the read fails with the error of the fallback. The review text lists each fallback read
  with its target, its trigger, and its fill fields. The triggers `empty` and `status` are owner proposals that the
  owner can still veto at review.
- A bundled operation of a connector spec takes `catalog_id` with the operation row of the built-in catalog that it
  serves, such as `documents.get` (ADR 002, E9). The spec module has two entry points: `Parse` serves every file of
  the user folder and refuses a built-in kind and the `catalog_id` field, and `ParseBundled` accepts both for the
  loader of the embedded specs only. Each `catalog_id` names one existing row of the same built-in kind, once, and an
  operation that no row names sets `cli: false` and `mcp: false`. The catalog row stays the source of the CLI name,
  the MCP tool name, the flags, the descriptions, the input schema, and the hints. The review text lists each bundled
  operation with its catalog row. E9 carries no spec yet.

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
- `retriever-spec` and `retriever-engine` are at `v0.6.0`, and the other libraries stay at `v0.4.0`. The connector spec
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
