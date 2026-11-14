---
title: retriever commands
description: The commands, options, and flags of the retriever command-line tool, as the binary prints them.
category: reference
tags:
  - reference
  - cli
  - retriever
---

# retriever commands

`retriever` is the sourcing command-line tool in `apps/retriever-cli`. It reads and, where a source allows it,
writes content of configured sources: Outline wikis, Kiwi test management, and Jira Cloud. This page lists the
commands. The binary prints the same facts with `retriever --help` and `retriever <command> --help`.

It is alpha. Breaking changes are allowed, and each one is listed in the [changelog](../../CHANGELOG.md).

## Synopsis

```text
retriever [global options] <command> [command options]
retriever help <command>
retriever --version
```

## Global options

- `-o, --output <format>`: `text` (default) or `json`. JSON prints one document with a `schemaVersion` field. The
  option is accepted before or after the command name.
- `--color <when>`: `auto` (default), `always`, or `never`. Output never contains escape sequences, so the value changes
  nothing today. No colour is emitted yet, so the `NO_COLOR` environment variable has no effect: a test checks that its
  set and empty values leave the output unchanged.
- `--catalog <path>`: the sources catalog file. See [configuration](./retriever-cli-configuration.md).
- `--env-file <path>`: the local env file that holds source keys. See [configuration](./retriever-cli-configuration.md).
- `--connectors-dir <path>`: the folder of connector specs. A loaded spec adds a command group named by its kind, such as
  `example-wiki docs list`. See [configuration](./retriever-cli-configuration.md) and
  [add a spec connector](../how-to/add-a-spec-connector.md).
- `-V, --version`: print `retriever <version>` and the alpha notice, then exit `0`.
- `-h, --help`: show help for the command it follows. Help wins over other options.

`--version` prints two lines, for example `retriever 0.2.0` and `ALPHA: expect breaking changes`.

`-v`, `-q`, and `--no-color` do not exist. Passing one is a usage error with exit status `2`.

## Command groups

- `sources`: lists the configured sources. `sources list` is the only subcommand.
- `document`: reads one document of any source kind. `document get` is the only subcommand.
- `outline`: Outline wiki commands. About 35 groups, such as `collections`, `documents`, and `export`.
- `kiwi`: Kiwi test-management commands: `export`, `products`, `documents`, `test-cases`, and `plans`.
- `jira`: Jira Cloud commands: `export`, `whoami`, `projects`, `issues`, and `comments`.
- `serve`: serves the read-only GraphQL API on the loopback address.
- `mcp`: serves the source catalog to a Model Context Protocol client over standard input and output.
- `help`: prints the help page of a command, such as `retriever help sources list`.

For the full list of `outline` groups and their subcommands, run `retriever outline --help` and then
`retriever outline <group> --help`. The list is generated from the operation catalog, so this page does not copy it.

## Changes and API tokens

A command that changes a source runs the change that it names. `retriever` does not check an access level, and it asks
for no confirmation flag first. The API token of each source decides what that source allows, so give each token only
the rights that you want the tool to use. [Who guards what](../explanation/who-guards-what.md) explains the split.

## Source selection

Commands that read a source accept `--source <id>` and, where the kind is not implied, `--kind <kind>`. The kind is
`outline`, `kiwi`, or `jira`. When more than one source of the kind exists, `--source` is required. Without it the
command fails with `retriever.<area>.ambiguous_source`.

## Commands in detail

### sources list

```text
retriever sources list [--kind <kind>] [--tag <tag[,tag...]>]
```

Lists the configured sources, sorted by source ID. `--kind` keeps one kind. `--tag` keeps the sources that have every
tag given, as a comma-separated list. In text mode the first line is the count, then one row per source.
In JSON mode the
document is described in [JSON schemas](./retriever-cli-json-schemas.md).

### document get

```text
retriever document get <documentId> [--kind <kind>] [--source <id>]
```

Prints one document. In text mode it prints the Markdown body. In JSON mode it prints `schemaVersion`, `id`, `title`,
`url`, `markdown`, and `collectionId`. `--kind` defaults to `outline`.

### outline collections

```text
retriever outline collections [--source <id>]
```

Lists the collections of an Outline wiki.

### outline documents list

```text
retriever outline documents list [--collection-id <id>] [--offset <n>] [--limit <n>] [--source <id>]
```

Lists one page of documents. Use `--offset` with the `next_offset` value that the previous page reported.

### outline documents search

```text
retriever outline documents search <query> [--offset <n>] [--limit <n>] [--source <id>]
```

Searches documents by title and text, one page at a time. `--limit` takes `1` to `100` and defaults to `25`. This is a
search command, so an empty result exits `1`. See [exit codes](./retriever-cli-exit-codes.md).

### outline export

```text
retriever outline export <documentId> --out <dir> [--source <id>]
```

Writes one Outline document as a Markdown file into the folder named by `--out`. The folder is created when it is
missing. The file is named by the document ID, `<documentId>.md`, with no title in the name. The file is plain Markdown
with no front matter. A file of the same name is always replaced: the write goes through a temporary file and a rename,
and no check for a collision runs. Text mode prints `Wrote <path>`. JSON mode prints `schemaVersion` and `location`.

### kiwi export and jira export

```text
retriever kiwi export <documentId> --out <dir> [--source <id>]
retriever jira export <documentId> --out <dir> [--source <id>]
```

The same export as `outline export`, for a Kiwi or a Jira document.

### serve

```text
retriever serve [--port <port>]
```

Serves the GraphQL API on `127.0.0.1` only. Port `0` (the default) picks a free port. The first line on standard output
names the address: `retriever serve listening on 127.0.0.1:<port>`. The API is read-only. It exposes the `sources`,
`collections`, `collectionTree`, and `document` queries and has no mutation. Introspection is off.

A request is checked in this order, and a refused request never reaches a source. The `Host` header must name a loopback
host with the served port. The method must be `POST`, and the content type must be JSON. A request that carries an
`Origin` header is refused, and the server sends no `Access-Control-` header. The body is capped at 1 MiB. A query over
the depth, complexity, or size limit is refused before any source is called. The served schema is described in
[the sourcing layer](../explanation/sourcing-layer.md).

`serve` stops on `SIGINT` or `SIGTERM` after a graceful shutdown. It exits `130` on `SIGINT` and `143` on `SIGTERM`, the
same statuses as `mcp`. It exits `0` when its server ends without a signal.

### mcp

```text
retriever mcp [--tools <resource,...>]
```

Runs one Model Context Protocol session over standard input and standard output. The session offers every tool of the
catalog, each with its read-only and destructive hints. `--tools` keeps the tools of the named resources, such as
`documents` or `comments`. The `list_sources` tool is always offered. The `--max-access` flag was removed, and passing
it exits `2`. Standard output carries only the protocol. Each tool call writes
one log line to standard error. The session ends when the client closes its input (exit `0`), on `SIGINT` (exit `130`),
or on `SIGTERM` (exit `143`).

## Related

- [Exit codes](./retriever-cli-exit-codes.md) lists every status a command can return.
- [JSON schemas](./retriever-cli-json-schemas.md) lists the fields of each JSON document and the error codes.
- [Configuration](./retriever-cli-configuration.md) gives the precedence of the catalog and env-file paths.
- [Environment variables](./environment-variables.md) lists the variable names.
- [Catalog format](./retriever-cli-catalog-format.md) describes the sources file.
- [Migrating from Butler](./retriever-cli-migration-from-butler.md) maps the old commands and flags.
