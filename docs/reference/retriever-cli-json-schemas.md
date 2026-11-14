---
title: retriever JSON schemas
description: The JSON documents that the retriever tool prints with --output json, and its failure body.
category: reference
tags:
  - reference
  - cli
  - retriever
  - json
---

# retriever JSON schemas

With `--output json`, `retriever` prints one JSON document per command on standard output. Every document carries
`schemaVersion`. In this release it is `2` for `sources list` and `1` for every other document. A caller may branch on
it. JSON output never contains escape sequences and never contains progress text.

Fields are added compatibly. A field is not removed, renamed, or given a new meaning under the same version. A breaking
change raises the version and is listed in the [changelog](../../CHANGELOG.md).

Select the mode with `--output json` or `-o json`, before or after the command name. The mode is never chosen
by checking
whether output is a terminal.

## Success documents

### sources list

```json
{
  "schemaVersion": 2,
  "sources": [
    { "id": "example-jira", "kind": "jira", "tags": ["tickets"] },
    { "id": "example-kiwi", "kind": "kiwi", "tags": ["tests"] }
  ]
}
```

- `sources`: an array with one object per configured source, sorted by `id`.
- `id`: a string, the source ID from the catalog.
- `kind`: a string, `outline`, `kiwi`, or `jira`.
- `tags`: an array of the source's tags. It is empty when the source has none.

The endpoint and the credential names are never printed.

The version of this document is `2`. Removing the `access` field is a breaking change, and a breaking change raises the
version. The other documents keep version `1`.

### document get

```json
{
  "schemaVersion": 1,
  "id": "doc-1",
  "title": "Example document",
  "url": "https://docs.example.test/doc/example-document",
  "markdown": "# Example document\n",
  "collectionId": "col-1"
}
```

The field order is fixed: `schemaVersion`, `id`, `title`, `url`, `markdown`, `collectionId`. These are the same fields
that the GraphQL `document` query returns.

### List reads

Each list command, such as `outline collections` or `outline documents list`, prints these fields:

- `schemaVersion`: always present, and `1`.
- `count`: always present. It is the number of items in this page.
- The list key: always present. It holds the array of items, and its name comes from the command, such as
  `collections` or `documents`.
- `has_more`: present when the command is paged. It is `true` when another page may follow.
- `next_offset`: present when `has_more` is `true`. It is the `--offset` value that reads the next page.
- `truncated`: present when the command names a limit in a notice. It is `true` when the list was cut at the limit.
  The command uses it in place of `has_more`, because such a list has no offset to continue from.

Each item is an object whose fields are the fields of the command's catalog row. Nested records stay nested.

### Object reads

A command that reads one object prints `schemaVersion` followed by the object's fields. The fields are listed in the
help of the command's catalog row, and the exact set is the one the source returns.

### export

```json
{ "schemaVersion": 1, "location": "/path/to/folder/doc-1.md" }
```

`location` is the path of the file the export wrote. It is the only field. The file is named by the document ID, so the
name holds no title.

## The failure body

When a command fails in JSON mode, standard output stays empty and standard error gets one JSON line:

```json
{
  "schemaVersion": 1,
  "command": "retriever outline documents get",
  "exitCode": 2,
  "error": {
    "code": "retriever.outline.key_missing",
    "message": "EXAMPLE_OUTLINE_API_KEY is not set",
    "retryable": false
  }
}
```

- `schemaVersion`: an integer, `1`.
- `command`: a string, the full command path that ran, such as `retriever sources list`.
- `exitCode`: an integer, the exit status the process returns. See [exit codes](./retriever-cli-exit-codes.md).
- `error.code`: a string, `retriever.<area>.<reason>`. The closed list is below.
- `error.message`: a string, one sentence for a person. It names a variable or a flag, never a value.
- `error.retryable`: a boolean. It is `true` for a rate limit, a timeout, or a network failure before any reply.

The CLI does not emit `error.field`. The convention makes it optional, and the CLI omits it so that rejected input is
never echoed.

Text mode writes the same failure as one line on standard error: `retriever: <message>`.

## Error codes

The codes are `retriever.<area>.<reason>`. The area is the command group, such as `outline`, `kiwi`, `jira`, `sources`,
`document`, `serve`, `mcp`, or `usage` for a mistake the parser found before any command ran. An error that names a
source kind takes that kind as its area.

Reasons that the CLI can emit:

- `failed`: a failure with no more specific reason.
- `refused`: the environment refused the start, such as a busy loopback port.
- `key_missing`: a source's key is not in the process environment or the env file.
- `identity_invalid`: a source's account e-mail variable holds a value that is not an address.
- `unknown_source`: no configured source has the ID given.
- `ambiguous_source`: more than one source of the kind exists, and no `--source` was given.
- `operation_unavailable`: the Outline server does not offer the endpoint, often because its version is older.
- `not_configured`: the composition root did not wire the command.
- `usage_invalid_argument`: a value is not valid for its option.
- `usage_unrecognized_argument`: an argument or option is not known.
- `usage_missing_argument`: a required argument or option is absent.
- `usage_invalid_output`: `--output` is not `text` or `json`.
- `usage_invalid_color`: `--color` is not `auto`, `always`, or `never`.
- `usage_no_subcommand`: a group was named with no subcommand.
- `invalid_*`: a typed value failed its rule, such as `invalid_uuid`, `invalid_url`, or `invalid_enum`.
- `source_required`: a change does not name its source with `--source`.
- `document_conflict`: a write was refused because the document changed since it was read.
- `unresolved_name`: a name that a write takes matches no entry, or more than one, in the source. The write sends
  nothing.
- `write_outcome_unknown`: a write's request may have reached the source, but its reply is missing or unusable. The
  change may or may not have happened, so the write is never retried. The command exits 2.
- `panic`: an internal failure. See [exit codes](./retriever-cli-exit-codes.md#panics).

The list is a summary. The `invalid_*`, `*_required`, and `*_mismatch` entries stand for groups of reasons, and the
code defines each one.

The reason list is closed. A new failure adds a reason, and an existing reason keeps its meaning.

## Related

- [Commands](./retriever-cli-commands.md) lists the commands that print these documents.
- [Exit codes](./retriever-cli-exit-codes.md) lists the statuses that accompany each code.
- [The sourcing layer](../explanation/sourcing-layer.md) explains why these documents are a contract.
