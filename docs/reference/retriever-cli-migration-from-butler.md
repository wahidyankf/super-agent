---
title: Migrating from Butler to retriever
description: The list that maps each Butler behaviour to its retriever counterpart, for callers of the old tool.
category: reference
tags:
  - reference
  - cli
  - retriever
  - migration
---

# Migrating from Butler to retriever

`retriever` is the sourcing tool that replaces the source-facing part of the Butler command-line tool. It is a new tool
with its own contract, so a caller that depends on Butler must change. This page lists each change. Read it before you
point a script, a harness, or a server at `retriever`.

Because the tool is alpha, a later release may change any row again. Each change is also listed in the
[changelog](../../CHANGELOG.md).

## The changes

Each entry gives the Butler behaviour first, then the retriever behaviour.

- **Binary.** Butler: `butler`. retriever: `retriever`.
- **Environment.** Butler: `BUTLER_*` variables, such as `BUTLER_RO_API_KEY`. retriever: `RETRIEVER_*` variables.
  Credential variable names come from each catalog entry.
- **Configuration.** Butler: a `--config` flag. retriever: `--catalog`, `--env-file`, `RETRIEVER_CATALOG`,
  `RETRIEVER_ENV_FILE`, then `./sources.yml`, `./.env.local`, and `~/.config/retriever`. See
  [configuration](./retriever-cli-configuration.md).
- **Built-in source.** Butler: a built-in private source. retriever: none. Every source comes from the catalog. A
  missing catalog exits `2`.
- **Output.** Butler: `--output` with `text`, `json`, or `markdown`. retriever: `--output text` or `json`. Markdown is
  removed.
- **Verbosity.** Butler: `-v`, `--verbose`, `-q`, and `--quiet`. retriever: removed. Passing any of them is a usage
  error with exit status `2`.
- **Colour.** Butler: `--no-color`. retriever: `--no-color` is removed. `--color auto`, `always`, or `never` is kept.
- **Usage errors.** Butler: exit `2` for `outline`, and exit `1` for `kiwi` and `jira`. retriever: exit `2`
  everywhere.
- **Failures.** Butler: exit `2` for `outline`, `kiwi`, and `jira`. retriever: exit `2`. A bounded wait that elapsed
  exits `124`. An environment refusal exits `125`.
- **Exit `1`.** Butler: also used for usage errors in `kiwi` and `jira`. retriever: only for a search that matched
  nothing. See [exit codes](./retriever-cli-exit-codes.md).
- **Failure body.** Butler: a JSON error body on standard error for `outline` with `--output json`. Not the same shape
  on every command. retriever: one JSON body on standard error for every command, with `schemaVersion`, `command`,
  `exitCode`, and `error`. See [JSON schemas](./retriever-cli-json-schemas.md).
- **Error codes.** Butler: not the `retriever.<area>.<reason>` form. retriever: `retriever.<area>.<reason>`, a closed
  list.
- **Export.** Butler: a vault export that needed a vault layout. retriever: `outline export`, `kiwi export`, and
  `jira export` need `--out <dir>`. The file is plain Markdown with no front matter. It is named by the document ID, not
  by a title, and a file of the same name is always replaced, with no collision check. JSON mode prints `schemaVersion`
  and `location`.
- **GraphQL export.** Butler: a write operation on the GraphQL server that exported a document. retriever: no mutation.
  `serve` is read-only, so a caller runs `outline export`, `kiwi export`, or `jira export` and reads the file it writes.
- **Credentials.** Butler: names such as `BUTLER_RO_API_KEY`. retriever: a key variable ends in `_API_KEY` or
  `_API_TOKEN`. Jira's account address ends in `_EMAIL`.
- **Removed commands.** Butler: the job report, PDF, GitHub, and vault commands, which stay in Butler. retriever: not
  present. They stay in the other tool that owns them.
- **New.** Butler: none. retriever: `document get`, the `help` subcommand, `sources list --output json`, `--version`
  that prints the name, `0.1.0`, and the alpha notice, and a panic that exits `2` as `retriever.panic` with no stack
  trace.

## Departures from Butler

These are the places where retriever behaves differently on purpose, beyond a rename.

- **No confirmation gates.** Butler reads the target of a critical change before it checks `--confirm`. retriever has no
  `--confirm`, `--yes`, or `--dry-run`, so a change runs as the command names it. Each of those flags is an unrecognized
  argument, with exit `2`. A change still names its source with `--source`.
- **Missing catalog.** There is no built-in source. A missing catalog exits `2` with `retriever.<area>.catalog_missing`.
  For `sources list` the code is `retriever.sources.catalog_missing`.
- **Plain HTTP refused at load.** An endpoint that is not HTTPS is refused when the catalog loads, before any request.
  The failure code is `retriever.<kind>.https_required`, for example `retriever.outline.https_required`.
- **Unknown outcome of a Kiwi write.** A write that Kiwi answers with a server error is sent once and never retried. The
  change may already have applied, so retriever reports `retriever.kiwi.write_outcome_unknown` with `retryable` set to
  `false`.

## What to change in a caller

1. Rename the binary and the environment variables. `BUTLER_` becomes `RETRIEVER_`.
2. Move the catalog and the env file, or set `RETRIEVER_CATALOG` and `RETRIEVER_ENV_FILE`. Remove any reliance on a
   built-in source.
3. Read the exit status before you read the output. A status `1` now means only an empty search. A status `2` means the
   command did not complete, whatever the output says.
4. Read `error.code` from the JSON failure body, not the text. Branch on `error.retryable` for `124` and for network
   failures.
5. Drop `-v`, `-q`, and `--no-color`. Use `--color never` if you need plain output.
6. For an export, pass `--out <dir>`, and expect a plain Markdown file named by the document ID. A script that looked for
   a file by its title must look for `<id>.md` instead.
7. For a dashboard that reads content, use `retriever serve`. It is read only.

## Related

- [Commands](./retriever-cli-commands.md) lists the commands that remain.
- [Configuration](./retriever-cli-configuration.md) gives the path precedence.
- [Environment variables](./environment-variables.md) lists the variable names.
