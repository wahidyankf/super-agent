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
that a schema cannot state. The `retriever` command loads each spec of its connectors folder through this module, and the
[connector engine](./connector-engine.md) runs it. The reasons are in
[ADR 001](../explanation/decisions/001-user-defined-connectors-and-profiles.md).

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
| `method`     | yes      | `GET` or `HEAD`, or `POST` with `read_over_post`. See POST reads.                   |
| `path`       | yes      | Under `base_path`. A `{name}` slot fills one segment. No segment starts with a dot. |
| `params`     | no       | Typed parameters. See below.                                                        |
| `headers`    | no       | A constant, or the whole `{name}` of a parameter. See below.                        |
| `body`       | no       | A JSON object. `{{name}}`, as a whole string, takes a body parameter at any depth.  |
| `response`   | yes      | How the reply maps to output fields or to Markdown. See below.                      |
| `pagination` | no       | `style`, the parameter names, and `max_pages`. See below.                           |
| `role`       | no       | `list-collections`, `collection-tree`, or `get-document`, for the gateway reads.    |
| `cli`, `mcp` | no       | A boolean, default `true`, or a custom name, which turns the surface on.            |
| `examples`   | yes      | At least one. See below.                                                            |

## Access and method

The `access` label and the `method` must form one of these pairs. Any other pair fails validation.

- A read takes `GET` or `HEAD`, and no `read_over_post` marker.
- A read takes `POST` only with `read_over_post: true`. The marker is refused on any other pair.
- A write or an admin operation takes `POST`, `PUT`, `PATCH`, or `DELETE`, and no marker. An admin operation with `GET`
  is refused.

## POST reads

A read may send its request as `POST` when the operation sets `read_over_post: true`, which is the only value the field
takes. The request is a read: it is not sent as a write, and it has no side effect that the engine knows of. The spec
must meet these rules, or it is refused with `retriever.connector.spec_invalid`:

- The operation has a `body`, a JSON object. `{}` is allowed. Each value is a constant or a whole `{{name}}` slot of an
  `in: body` parameter, at any depth.
- `headers` may not set `Content-Type`. The engine sends `application/json` for a POST read.
- No parameter has `resolve`. A POST read claims no side effect, and a resolve would be one.
- The body that is rendered is at most 64 KiB and nests at most 8 levels, counting the top-level object as level 1. A
  body over either limit fails at run time with `retriever.<kind>.body_limit`, before any request. A spec field does
  not change these limits.

The review text lists each POST read in its own block, with its path and the slots that its body takes.

## Parameters

Each parameter has `name`, `type`, `in` (`path`, `query`, or `body`), and `required`. It may have `default`, `min`,
`max`, `values`, `help`, and `resolve`. The `type` is one of the 21 parameter kinds of the operation catalog, such as
`ID`, `Query`, `Limit`, `Enum`, `Time`, and `URL`, so every adapter validates a value the same way. Two rules narrow it:

- A path parameter has type `ID`, `UUID`, or `ShareID`, and it is always required. The path must use it.
- A query parameter may not have a credential-shaped name, such as `api_key`, `token`, `secret`, or `password`.

Each name is unique within its operation. An `Enum` or `EnumList` parameter lists its `values`.

`resolve` turns a name into an ID before a write. It names a read operation of the same spec and one of its response
fields. The spec module checks those names. The engine, in `libs/retriever-engine`, runs the read once, with the name
under the parameter of the field's own name when the read declares one. It matches the entries whose field equals the
name exactly, and it sends the write only when exactly one entry matches. Its `id` field is the ID. The read must be an
unpaged list. A name that matches zero or two entries fails with `retriever.<kind>.unresolved_name`. This replaces the
Kiwi name lookups.

## Headers

`headers` maps each header name to one of two values: a constant of at most 256 printable characters, or the whole
slot `{name}` of a parameter. A value built from a parameter, such as `tenant-{id}`, is refused. The header names
`Authorization`, `Cookie`, `Host`, and `Content-Length` are refused here, without regard to case, because the engine
alone sets them. The header parameter location of the draft is not in the spec: a header comes from `headers`.

## Pagination

- `style`: `offset`, `cursor`, or `none`.
- `in`: `query` (the default) or `body`. With `in: body`, every parameter the paging rule names must be a declared
  body parameter of the same operation; a rule cannot mix locations.
- `max_pages`: the page cap of the read, from 1 to 20. It is the stop rule, so `offset` and `cursor` always name it.
- `offset`: needs `limit_param` and `offset_param`. With `in: query` both name query parameters of the operation; with
  `in: body` both name body parameters, and the values travel in the JSON body instead of the query. The offset is a
  non-negative integer below 2^31. The engine advances it by the number of items in the page, and stops at the first
  short page. At the cap it returns the pages it has, with a continuation that names the next offset.
- `cursor`: needs `cursor_param` and `next_pointer`, a JSON pointer in the reply. The location rule is the same: with
  `in: body` the cursor parameter is a body parameter, and the value travels in the JSON body. A next cursor that
  repeats one already sent ends the read, so a reply that never advances cannot run to the cap. At the cap the engine
  returns the pages it has, with no continuation.
- `none`: takes no other field.

The review text lists each body-paged read in its own block, with its style, its body parameters, and its page cap.

## Response

- `items`: a JSON pointer to the list, for a list operation.
- `first`: a JSON pointer to the list whose first item a read maps, for a first-item read such as `shares.get`.
  The engine takes item 0 and maps its fields with pointers relative to the item; only index 0 exists. An empty
  list ends with `retriever.<kind>.not_found` and no partial record, and a value that is not a list ends with
  `unexpected_response`. A response holds at most one of `items`, `first`, and `merge`.
- `merge`: two to eight lists merged into one output, for a read such as `suggestions.mention`, in place of `fields`
  and of `items` and `first`. It names `label_field`, the output field that carries the constant label of each list,
  and `lists`, each with `items` (a JSON pointer to the list in the reply), `label` (a constant of lowercase letters
  and hyphens, at most 32 characters), and `fields` (the output fields of that list, with the same rules as above,
  including `required`, `format`, and one level of `object`). The output keeps the declared list order, then the reply
  order, and it shares the output cap of any list. The label beats a reply field of the same name. A pointer that
  names no list, and an item that is not an object, end with `unexpected_response`.
- `tree`: a tree read that flattens the nested nodes of the `items` list depth first, in place of `first` and of
  `merge`. It names `children` (a JSON pointer to the child nodes of one node, read relative to the node),
  `parent_field` and `depth_field` (the output fields that carry the parent and the depth of each node), an optional
  `root_parent_param` (the read parameter whose value is the parent of the top nodes, null when it names none), and
  an optional `max_depth` (the deepest node the read keeps, 64 when unset, and never above 64). The parent of a node
  is the id of its parent node, so the response holds a required `id` field; the depth counts from 0. A node deeper
  than the cap, and its subtree, is left out. The computed fields beat a mapped reply field of the same name. A
  reply with `tree` decodes to a fixed higher cap of 146 levels, which covers a tree of 70 levels; no spec field sets
  it, and no other reply changes.
- `fields`: a map from output field to an object with `pointer`, a JSON pointer into the reply, an optional `required`,
  an optional closed `format`, and an optional `convert`. The converter is `plain`, `markdown-from-html`, or
  `markdown-from-adf`. The engine does not serve the two Markdown converters yet. A spec that names one still loads,
  and each run of that operation fails with `retriever.<kind>.unsupported_operation` before any request is sent.
- `required`: `true` on a reply field fails the read with `unexpected_response` when the reply holds a missing, null,
  or empty value for it, and no partial record is returned. A missing optional field still maps to null.
- `format`: one closed name, `id`, `uuid`, or `share-id`, checked on the value with the same checks the catalog kinds
  use on request parameters. A free pattern has no form in the language. The failure names the field and never the
  value.
- `object`: a reply field may hold one nested record instead of a scalar: `user` with `object` holding at most 32
  named sub-fields, each a pointer with its own optional `required`, `format`, and `convert`. One level only: a
  sub-field with its own `object` is refused, and so is an object field with a scalar keyword of its own. A pointer
  that reaches an object or a list is refused, so a whole object is never copied. The mapper reads declared pointers
  only, and the reply tree is dropped after mapping. A failure names the dotted path, never the value.
- `markdown`: the output field that holds the Markdown body, for a document read. It must name a field.
- `errors`: a map from HTTP status, from 100 to 599, to a reason from the closed list in the error section.

The review text lists each operation with required or checked fields in its own block, with each required field and
each checked field with its closed format name. The review text lists each operation with a nested record field in
its own block, with the object field and each sub-field. The review text lists each first-item read in its own
block, with the operation and the list pointer as `first of /data`. The review text lists each merge read in its own
block, with the label field and each list as `<label> of <pointer>`, and it joins the required, checked, and nested
fields of every list under the same operation. The review text lists each tree read in its own block, with the items
pointer, the children pointer, the computed fields, the root parent parameter when the read names one, and the depth
cap.

There are no expressions. A pointer reads, and nothing computes.

## Examples

Each example holds `request` (parameter values), `response` (status, optional constant headers, and body), and `expect`
(the output fields that the reply must map to). The `request` keys must name parameters of the operation, and the
`expect` keys must name its output fields. A reply may not carry an `Authorization` header, and no string of an example
may hold a bearer or basic credential or a token-shaped run of 32 characters. A spec holds no host, so an example names
none. The bundled fixture uses `example.test` names only.

## Limits for the Outline reads

The language cannot yet express the Outline reads, so they still run on the Go gateway. Four limits block them:

- Outline sends every call as a `POST` with a JSON body. A read may now use `POST` with `read_over_post`, which is
  described in [POST reads](#post-reads), so this limit no longer blocks a read.
- Outline paging puts `offset` and `limit` in the body. A paging rule may now travel in the body with `in: body`,
  which is described in [Pagination](#pagination), so this limit no longer blocks a read.
- A field is never required. A field takes `required` and a closed `format` now, which is described in
  [Response](#response), so this limit no longer blocks a read. A missing optional field still maps to null.
- A field holds one scalar or one nested record, which is described in [Response](#response). The first item of a
  list, a list merged from several lists, and a tree with a computed depth are described in [Response](#response); a
  fallback read has no form in the language.

A spec named `outline` is also refused, because the kind is a built-in source. A bundled Outline spec would need an
exception for bundled files only.

## Error reasons

A refused spec carries the stable code `retriever.connector.spec_invalid`, which the spec module defines. The app reports it
with exit status `2`, and the failure names the file and the field path. A catalog `spec_sha256` that does not match the
file reports `retriever.connector.spec_pin_mismatch`, and a connectors folder that `--connectors-dir` names but cannot use
reports `retriever.connector.folder_unavailable`. The draft's other connector reasons, `plugin_pin_mismatch` and
`plugin_protocol`, belong to later changes, and this module does not define them. Runtime failures use `retriever.<kind>.<reason>` with the reasons of the
[error codes](./retriever-cli-json-schemas.md#error-codes). A write whose outcome is unknown uses
`write_outcome_unknown`, which exits 2 and is never retried. Exit statuses follow the
[exit-code contract](./retriever-cli-exit-codes.md).

## Planned extensions

[ADR 002](../explanation/decisions/002-spec-language-extensions.md) plans nine closed extensions of this language, so
that the Outline, Kiwi, and Jira connectors can run as specs. The first seven of them, POST reads (E1), body paging
(E2), required and validated fields (E3), nested records with an allow-list (E4), the first item or not found
(E5), merged lists with constant labels (E6), and computed tree fields (E7), are built and described above. The
other two are not built. The
limits of the Outline reads, above, describe
the language before ADR 002 for the
shapes that the remaining extensions cover. This page documents only what the module accepts today, and each extension
moves here when its pull request lands.
