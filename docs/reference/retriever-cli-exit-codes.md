---
title: retriever exit codes
description: Every exit status that the retriever tool returns, what it means, and which commands return it.
category: reference
tags:
  - reference
  - cli
  - retriever
  - exit-status
---

# retriever exit codes

Exit statuses are a contract. A script or a harness may branch on them, so the set is closed. A new failure gets a new
error code in the [JSON error body](./retriever-cli-json-schemas.md), never a new number. Every command's `--help` page
lists the statuses it can return.

## The statuses

- `0`: the command completed, and the result is affirmative. A retry is not needed.
- `1`: a search completed and matched nothing. The empty result is still printed. This is not a failure.
- `2`: the command did not complete. The cause is a usage mistake, a configuration error, a failed request, or an
  internal failure. Retry only after a fix.
- `124`: a bounded wait elapsed before the command finished, such as a rate limit or a timeout. A retry is allowed.
- `125`: the environment refused before the command started. The cause is a loopback port that another process holds, or
  a path that the environment cannot name. Retry only after a change.
- `128+N`: the process was ended by signal `N`. `130` is an interrupt, and `143` is `SIGTERM`. Whether to retry depends
  on the caller.

The CLI returns `0`, `1`, `2`, `124`, and `125` from its own logic. It returns `130` and `143` only from
[`retriever mcp`](./retriever-cli-commands.md#mcp), which reports an interrupt and `SIGTERM` as explicit statuses. Any
other signal that ends the process shows up as `128+N` in the shell.

## Which commands return which statuses

- **Every command** can return `0`, `2`, `124`, and `125`.
- **Search reads** also return `1` when the result is empty. Today these are `outline documents search` and
  `outline documents search-titles`. Their help pages say so. An empty result is printed on standard output before the
  status is set, so a caller can read it.
- **`retriever mcp`** returns `0` when the client closes its input, `2` for a usage or setup error, `130` on an
  interrupt, and `143` on `SIGTERM`.
- **`retriever serve`** returns `0` after a shutdown caused by `SIGINT` or `SIGTERM`, and `125` when the loopback port
  cannot be bound. Its help page lists `0`, `2`, `124`, and `125`.

The CLI never returns `126` or `127`, because it starts no child process.

## Usage and configuration errors

Every usage mistake exits `2`. This includes an unknown option, a missing argument, a bad `--output` or `--color` value,
an unknown source kind, a port outside `0` to `65535`, and a catalog that does not load. The message goes to standard
error, and standard output stays empty. A missing catalog is the same case. It exits `2` and names the three ways to
point at a catalog.

## Panics

An internal panic exits `2` through a top-level handler. The handler prints no stack trace. In JSON mode it writes a
failure body with the code `retriever.panic`. In text mode it writes one line that ends with `(retriever.panic)`.

## Signals and closed pipes

`serve` catches `SIGINT` and `SIGTERM` and shuts down cleanly, so it exits `0`. `mcp` reports the same two signals as
`130` and `143`, because it ends its session rather than shutting down a listener. The two commands therefore differ on
an interrupt. This difference is recorded as an open question in the changelog.

A closed standard output pipe relies on the Go runtime's default behaviour, which ends the process by `SIGPIPE`. A shell
reports that as `141`. No test in the repository checks this path.

## Related

- [Commands](./retriever-cli-commands.md) lists the commands and their flags.
- [JSON schemas](./retriever-cli-json-schemas.md) lists the error body and its codes.
- [Migrating from Butler](./retriever-cli-migration-from-butler.md) maps the old exit codes to the new ones.
