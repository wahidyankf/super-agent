---
title: Connector spec fields
description: >-
  The fields of a declarative connector spec, version 1, and the rules the spec module checks for each one.
category: reference
tags:
  - reference
  - connectors
  - spec
---

# Connector spec fields

**Status: as built in `libs/retriever-spec`.** The module checks a document against the JSON Schema
[connector-spec-v1.schema.json](../../libs/retriever-spec/schema/connector-spec-v1.schema.json), then against the rules
that a schema cannot state. No command loads a spec yet, so the rules here bind the module, not a running engine. The
reasons are in [ADR 001](../explanation/decisions/001-user-defined-connectors-and-profiles.md).

A spec is one YAML document of at most 256 KiB. It holds names and rules, never a value. An unknown field is an error.
The reader refuses an alias, an anchor, a repeated field name, a tag other than the plain forms of text, number,
boolean, null, map, and list, and a number that is not finite. A field name that is not a plain identifier is withheld
from every finding path.

## Top-level fields

| Field        | Required | Meaning                                                                  |
| ------------ | -------- | ------------------------------------------------------------------------ |
| `version`    | yes      | Must be `1`.                                                             |
| `kind`       | yes      | A lower-case slug. It must not name a built-in source.                   |
| `summary`    | yes      | One line of help text, at most 200 characters, with no braces.           |
| `endpoint`   | yes      | `base_path` (optional, fixed prefix) and `host_suffix` (optional, list). |
| `auth`       | yes      | How the key travels. See below.                                          |
| `limits`     | no       | Optional. Each limit lowers a cap: 30 s, 10 MiB, and 20 pages.           |
| `operations` | yes      | One to 200 operations.                                                   |

A spec holds no host. The catalog source names the host, and `host_suffix` only narrows which hosts may bind to the
spec. Each operation id starts with the `kind` of its spec, followed by a dot.

## Authentication

- `type`: `bearer`, `basic`, or `header`. The engine sets the header itself.
- `credential_env`: a variable name matching `^[A-Z][A-Z0-9_]*_API_(KEY|TOKEN)$`.
- `identity_env`: required for `basic`, and refused otherwise. It matches `^[A-Z][A-Z0-9_]*_EMAIL$`.
- `header_name`: required for `header`, and refused otherwise. `Authorization`, `Cookie`, `Host`, and `Content-Length`
  are refused here.

## Operation fields

| Field        | Required | Meaning                                                                             |
| ------------ | -------- | ----------------------------------------------------------------------------------- |
| `id`         | yes      | `<kind>.<group>.<verb>`, unique in the spec.                                        |
| `summary`    | yes      | One line for help and MCP descriptions.                                             |
| `access`     | yes      | `read`, `write`, or `admin`. A label, never a ceiling.                              |
| `method`     | yes      | A read takes `GET` or `HEAD`. A write takes `POST`, `PUT`, `PATCH`, or `DELETE`.    |
| `path`       | yes      | Under `base_path`. A `{name}` slot fills one segment. No segment starts with a dot. |
| `params`     | no       | Typed parameters. See below.                                                        |
| `headers`    | no       | A constant, or the whole `{name}` of a parameter. See below.                        |
| `body`       | no       | A JSON object. `{{name}}`, as a whole string, takes a body parameter at any depth.  |
| `response`   | yes      | How the reply maps to output fields or to Markdown. See below.                      |
| `pagination` | no       | `style`, the parameter names, and `max_pages`. See below.                           |
| `role`       | no       | `list-collections`, `collection-tree`, or `get-document`, for the gateway reads.    |
| `cli`, `mcp` | no       | A boolean, default `true`, or a custom name, which turns the surface on.            |
| `examples`   | yes      | At least one. See below.                                                            |

## Parameters

Each parameter has `name`, `type`, `in` (`path`, `query`, or `body`), and `required`. It may have `default`, `min`,
`max`, `values`, `help`, and `resolve`. The `type` is one of the 21 parameter kinds of the operation catalog, such as
`ID`, `Query`, `Limit`, `Enum`, `Time`, and `URL`, so every adapter validates a value the same way. Two rules narrow it:

- A path parameter has type `ID`, `UUID`, or `ShareID`, and it is always required. The path must use it.
- A query parameter may not have a credential-shaped name, such as `api_key`, `token`, `secret`, or `password`.

Each name is unique within its operation. An `Enum` or `EnumList` parameter lists its `values`.

`resolve` turns a name into an ID. It names a read operation of the same spec and one of its response fields. The spec
module checks those names. The engine, when it exists, must find exactly one match at run time. It replaces the Kiwi
name lookups.

## Headers

`headers` maps each header name to one of two values: a constant of at most 256 printable characters, or the whole
slot `{name}` of a parameter. A value built from a parameter, such as `tenant-{id}`, is refused. The header names
`Authorization`, `Cookie`, `Host`, and `Content-Length` are refused here, without regard to case, because the engine
alone sets them. The header parameter location of the draft is not in the spec: a header comes from `headers`.

## Pagination

- `style`: `offset`, `cursor`, or `none`.
- `max_pages`: the page cap of the read, from 1 to 20. It is the stop rule, so `offset` and `cursor` always name it.
- `offset`: needs `limit_param` and `offset_param`, both query parameters of the operation.
- `cursor`: needs `cursor_param`, a query parameter, and `next_pointer`, a JSON pointer in the reply.
- `none`: takes no other field.

## Response

- `items`: a JSON pointer to the list, for a list operation.
- `fields`: a map from output field to an object with `pointer`, a JSON pointer into the reply, and an optional
  `convert`. The converter is `plain`, `markdown-from-html`, or `markdown-from-adf`.
- `markdown`: the output field that holds the Markdown body, for a document read. It must name a field.
- `errors`: a map from HTTP status, from 100 to 599, to a reason from the closed list in the error section.

There are no expressions. A pointer reads, and nothing computes.

## Examples

Each example holds `request` (parameter values), `response` (status, optional constant headers, and body), and `expect`
(the output fields that the reply must map to). The `request` keys must name parameters of the operation, and the
`expect` keys must name its output fields. A reply may not carry an `Authorization` header, and no string of an example
may hold a bearer or basic credential or a token-shaped run of 32 characters. A spec holds no host, so an example names
none. The bundled fixture uses `example.test` names only.

## Error reasons

A refused spec carries the stable code `retriever.connector.spec_invalid`. The code is defined in the spec module. The
app does not map it to an exit status yet, so no command reports it. The draft's other connector reasons,
`spec_pin_mismatch`, `network_refused`, `plugin_pin_mismatch`, and `plugin_protocol`, belong to later changes, and this
module does not define them. Runtime failures use `retriever.<kind>.<reason>` with the reasons of the
[error codes](./retriever-cli-json-schemas.md#error-codes). Exit statuses follow the
[exit-code contract](./retriever-cli-exit-codes.md).
