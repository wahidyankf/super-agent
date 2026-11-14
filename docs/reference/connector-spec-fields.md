---
title: Connector spec fields
description: >-
  PLANNED. A draft of the fields of a declarative connector spec, so the implementation has a contract to build from.
category: reference
tags:
  - reference
  - planned
  - connectors
---

# Connector spec fields (planned)

**Status: PLANNED.** No spec loader, no engine, and no schema file exist yet. This page drafts the fields. It is not
the schema. The schema file arrives with the spec module, and the
[operation and spec JSON Schema](./operation-json-schema.md) page tracks it. The reasons are in
[ADR 001](../explanation/decisions/001-user-defined-connectors-and-profiles.md).

A spec is one YAML document of at most 256 KiB. It holds names and rules, never a value. An unknown field is an error.

## Top-level fields

| Field        | Required | Meaning                                                                          |
| ------------ | -------- | -------------------------------------------------------------------------------- |
| `version`    | yes      | Must be `1`.                                                                     |
| `kind`       | yes      | A lower-case slug. It names the catalog kind and the first word of CLI commands. |
| `summary`    | yes      | One line for help text and the README text derived from the spec.                |
| `endpoint`   | yes      | `base_path` (optional, fixed prefix) and `host_suffix` (optional, list).         |
| `auth`       | yes      | How the key travels. See below.                                                  |
| `limits`     | no       | `timeout_seconds`, `max_reply_bytes`, `max_pages`. Each may only lower the cap.  |
| `operations` | yes      | One to 200 operations.                                                           |

The host is never in a spec. The catalog source names it, as today. `host_suffix` only narrows which hosts may bind to
the spec.

## Authentication

- `type`: `bearer`, `basic`, or `header`. The engine sets the header itself.
- `credential_env`: a variable name matching `^[A-Z][A-Z0-9_]*_API_(KEY|TOKEN)$`.
- `identity_env`: required for `basic`, refused otherwise, matching `^[A-Z][A-Z0-9_]*_EMAIL$`.
- `header_name`: required for `header`. It is not a reserved name.

The spec cannot set `Authorization`, `Cookie`, `Host`, or `Content-Length`.

## Operation fields

| Field        | Required | Meaning                                                                                   |
| ------------ | -------- | ----------------------------------------------------------------------------------------- |
| `id`         | yes      | `<kind>.<group>.<verb>`, unique in the spec.                                              |
| `summary`    | yes      | One line for help and MCP descriptions.                                                   |
| `access`     | yes      | `read`, `write`, or `admin`. A label, never a ceiling.                                    |
| `method`     | yes      | `GET`, `HEAD`, `POST`, `PUT`, `PATCH`, or `DELETE`. DELETE needs `write` or `admin`.      |
| `path`       | yes      | A path under `base_path`. A `{name}` slot fills one whole segment.                        |
| `params`     | no       | Typed parameters. See below.                                                              |
| `body`       | no       | A JSON object. A string that is exactly `"{{name}}"` takes the typed value.               |
| `response`   | yes      | How the reply maps to output fields or to Markdown. See below.                            |
| `pagination` | no       | `style` (`offset`, `cursor`, or `none`), the parameter names, and `max_pages` (up to 20). |
| `role`       | no       | `list-collections`, `collection-tree`, or `get-document`, to serve the gateway reads.     |
| `cli`, `mcp` | no       | Booleans, default `true`. Each may carry a custom name.                                   |
| `examples`   | yes      | At least one. See below.                                                                  |

## Parameters

Each parameter has `name`, `type`, `in` (`path`, `query`, `body`, or `header`), and `required`. It may have `default`,
`min`, `max`, `values`, and `help`. The `type` is one of the existing parameter kinds of the operation catalog, such as
`ID`, `Query`, `Limit`, `Enum`, `Time`, and `URL`, so every adapter validates a value the same way.

`resolve` turns a name into an ID. It names a read operation of the same spec and the field to match, and it needs
exactly one match. It replaces the Kiwi name lookups.

## Response

- `items`: a JSON pointer to the list, for a list operation.
- `fields`: a map from output field to JSON pointer. An optional `convert` names a closed converter: `plain`,
  `markdown-from-html`, or `markdown-from-adf`.
- `markdown`: the field that holds the Markdown body, for a document read.
- `errors`: a map from HTTP status to a reason from the closed list below.

There are no expressions. A pointer reads, and nothing computes.

## Examples

Each example holds `request` (parameter values), `response` (status, optional headers, body), and `expect` (the mapped
fields). The conformance suite replays each one through the engine. An example may hold no credential-shaped value, and
its hosts are `example.test`, `example.invalid`, or `example.com`.

## Error reasons

New reasons use the area `connector`: `spec_invalid`, `spec_pin_mismatch`, `network_refused`, `plugin_pin_mismatch`, and
`plugin_protocol`. Runtime failures use `retriever.<kind>.<reason>` with the reasons of the
[error codes](./retriever-cli-json-schemas.md#error-codes). Exit statuses follow the
[exit-code contract](./retriever-cli-exit-codes.md).
