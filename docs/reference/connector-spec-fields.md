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
| `catalog_id` | no       | Only a bundled spec may name it: the catalog row the operation serves.              |
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

Each parameter has `name`, `type`, `in` (`path`, `query`, or `body`), and `required`. It may have `query_key`,
`default`, `min`, `max`, `values`, `help`, and `resolve`. The `type` is one of the 28 parameter kinds of the operation catalog, such as
`ID`, `Query`, `Limit`, `Enum`, `Time`, and `URL`, so every adapter validates a value the same way. A bundled port
also uses `CollectionID`, `CollectionUUID`, and `DocumentID`, which validate as an opaque ID or UUID, `IssueKey`,
which validates as the strict upper-case issue key with no lowercase, missing hyphen, leading zero, or traversal
text, and `Icon`, `URLList`, and `EventList`, which validate exactly as the catalog does: an icon is one emoji, a
named icon, or an attachment UUID, but never blank; a URL list holds 1 to 10 HTTPS addresses of at most 1024
characters each; an event list holds at most 50 event names, each `*` or a dotted name such as `documents.create`.
Names start lower-case and may hold upper-case letters, so a bundled operation names the catalog parameters exactly
(`collectionId`, `searchSource`). Two rules narrow the type:

- A path parameter has type `ID`, `UUID`, `ShareID`, or `IssueKey`, and it is always required. The path must use it.
  The engine binds `IssueKey` through the existing `KindIssueKey` validator, so a value outside the shape fails
  before any request with `invalid_document_id` for the path parameter, and the failure never echoes the value. Proof:
  `TestIssueKeyLowercaseIsInvalid`, `TestIssueKeyTraversalIsInvalid`, `TestIssueKeyRefusalNamesPath`, and
  `TestIssueKeyCanonicalAccepted`.
- A query parameter may not have a credential-shaped name, such as `api_key`, `token`, `secret`, or `password`.

Each name is unique within its operation. An `Enum` or `EnumList` parameter lists its `values`.

A query parameter may name the wire key it travels under with `query_key`, such as `limit` riding as `maxResults` and
`offset` as `startAt` for Jira. The catalog name stays the parameter name everywhere else: the CLI flag, the examples,
and the stub binding. The alias starts with a letter and holds letters, digits, and underscores only. An alias on a
parameter outside the query, one repeating the parameter name, and one repeating another query key of the same
operation are refused at load. The engine sends the query pair under the alias, including the offset, cursor, and
page-all query keys.

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

- `style`: `offset`, `cursor`, `page-all`, or `none`.
- `in`: `query` (the default) or `body`. With `in: body`, every parameter the paging rule names must be a declared
  body parameter of the same operation; a rule cannot mix locations. A `page-all` rule names no parameter, so the
  engine sends the fixed paging keys `offset` and `limit` itself, in the query or in the JSON body per `in`.
- `max_pages`: the page cap of the read, from 1 to 20. It is the stop rule, so `offset`, `cursor`, and `page-all`
  always name it.
- `page_size`: the fixed page size of a `page-all` read, from 1 to 100. Only a `page-all` rule names it.
- `offset`: needs `limit_param` and `offset_param`. With `in: query` both name query parameters of the operation; with
  `in: body` both name body parameters, and the values travel in the JSON body instead of the query. The offset is a
  non-negative integer below 2^31. The engine advances it by the number of items in the page, and stops at the first
  short page. At the cap it returns the pages it has, with a continuation that names the next offset. An offset read
  may declare the page-end rule instead: `last_pointer` (a JSON pointer in the reply to the flag that decides last)
  and `cut_to_limit` (a `Limit` or `Int` parameter whose value cuts the page). The reply must hold a bool at the last
  pointer or the read fails with `unexpected_response` naming the pointer path; more is the negation of that flag,
  beating the short-page rule, so a full page with `isLast` true claims no further page. A page holding more items
  than the cut keeps the first items and claims more, so no item is lost from the next page. Proof:
  `TestLastPointerFullPageIsLast`, `TestLastPointerMissingIsUnexpected`, and `TestCutToLimitKeepsFirstN`.
- `cursor`: needs `cursor_param` and `next_pointer`, a JSON pointer in the reply. The location rule is the same: with
  `in: body` the cursor parameter is a body parameter, and the value travels in the JSON body. A next cursor that
  repeats one already sent ends the read, so a reply that never advances cannot run to the cap. At the cap the engine
  returns the pages it has, with no continuation. A cursor rule that declares `last_pointer` may also declare the
  computed `truncated` flag; see the `truncated` entry under `Response`.
- `page-all`: reads every page of an operation that declares no page parameters, until a short page, with no
  continuation. The engine spaces the requests 100 ms apart, mirroring the pacing of the Go reader it replaces. At
  the cap it fails with the page-cap error instead of returning the pages it has. A `page-all` rule on a write is
  refused, as a write takes one request.
- `none`: takes no other field.

The review text lists each body-paged read in its own block, with its style, its body parameters, and its page cap. A
`page-all` read lists its fixed page size in place of parameters.

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
- `truncated`: the computed per-item flag of a cursor read with the page-end rule. It names `token` (the JSON pointer
  of the next-page token) and `last` (the JSON pointer of the last flag). Each item gains `truncated`, which is true
  exactly when the token is present and non-empty and last is false, or when the cut kept only the first items. A
  missing token counts as empty and a missing last counts as false, as the Go code reads them. A non-bool last or a
  non-string token fails with `unexpected_response` naming the path. The block is allowed only with a cursor rule that
  declares `last_pointer`; an example may expect `truncated` exactly then, like the computed parent and depth fields
  of a tree read. The flag beats a mapped reply field of the same name. Proof: `TestTruncatedNeedsTokenAndNotLast` and
  `TestTruncatedMissingTokenIsFalse`.
- `echo`: the key check of a read of one object. It names `param` (the request parameter whose value the reply must
  echo) and `pointer` (the JSON pointer of the echoed value in the reply). After mapping, the text at the pointer must
  equal the request argument exactly, or the read fails with `unexpected_response` naming the pointer path, never the
  values. Only a single-object read declares one: a list, first-item, merge, tree, paged, or write response is refused
  at load, and a missing, null, or non-text value at the pointer fails the same way at run time. Proof:
  `TestKeyEchoMismatchIsUnexpected`, `TestKeyEchoRefusalNamesPath`, and `TestKeyEchoOnListIsRefused`.
- `fallback`: a declared second read of one object, run when the first reply triggers it. It names `operation` (the
  ID of a read of the same spec, picked in the document and never from reply data), `when` (the one closed trigger:
  `empty` with a mapped output field that is empty, or `status` with a list of HTTP statuses), and `fill` (the
  output fields that take the target's values). The target declares only parameters the first operation also
  declares, and it receives their values. At most one fallback request runs per call, and the target has no fallback
  of its own, so a chain of two fallbacks is refused at load. Both requests share the 30 second limit and run under
  every safeguard of the engine: the same host, the same key, no redirect, and the same caps. When the fallback
  fails, the read fails with the error of the fallback. The triggers `empty` and `status` are owner proposals that
  the owner can still veto at review.
- `fields`: a map from output field to an object with `pointer`, a JSON pointer into the reply, an optional
  `fallback_pointer`, a second JSON pointer that is read when the first finds nothing, so one field maps two reply
  shapes, an optional `required`, an optional closed `format`, and an optional `convert`. A pointer that finds an
  explicit null wins, so the fallback serves only the shape that lacks the pointer, never a second guess inside one
  shape. A fallback pointer needs the pointer it falls back from. The converter is `plain`, `markdown-from-html`,
  `markdown-from-adf`, `hostname`, `text-from-editor-data`, `jira-time`, `jira-date`, or `jira-title`. The engine
  serves `hostname`: the field holds the host of the reply address, without the path, the query, the port, or the
  user name, or the empty text when the address has no host. The engine serves `text-from-editor-data`: the plain
  text of editor data, with a line break after each text block, or the raw text unchanged when the field falls back
  to it; a missing value maps to null, and any other scalar is an unexpected response. The engine serves
  `markdown-from-adf`: the Markdown of the ADF document, rendered as a pure function of the reply, ending with the
  truncation marker past the depth cap. The engine serves `jira-time`: the UTC instant of a Jira timestamp as RFC
  3339; `jira-date`: a calendar day unchanged; and `jira-title`: one line of at most 255 runes with no hidden
  character. Any other value, including a missing one, maps to the empty text with the field present, so the row is
  kept with a visible gap, exactly as the Go code answers today.
  A field may build its value from an issue key instead of reading it: `project_key` (the text before the last
  hyphen), `issue_id` (the key prefixed with `issue-`), `collection_id` (the project part prefixed with `project-`),
  and `browse_url` (the endpoint origin joined with `/browse/` and the key, never a URL the reply carries). Each names
  the JSON pointer of the key in the reply, all four share one closed issue-key check, and a missing, null, non-text,
  or misshaped value fails with `unexpected_response` naming the field, never the value. A built field reads one key
  only and takes no `pointer`, `fallback_pointer`, `format`, or `convert` of its own. Proof:
  `TestBrowseUrlIgnoresReplyUrl`, `TestBuiltIdRefusesBadKey`, and `TestProjectKeyTakesPrefix`. Proof: `TestJiraTimeMapsBadLayoutToEmpty`,
  `TestJiraTitleKeeps255Runes`, `TestJiraTitleStripsInvisible`, `TestJiraDateRefusesBareYear`,
  `TestAJiraTitleStripsABidiOverride`, and `TestAJiraBadDateMapsToEmptyWithTheFieldPresent`. The converter is a
  pure function of the reply: it sends no request and reaches no other host. The engine does not serve the
  `markdown-from-html` converter yet. A spec that names it still loads, and each
  run of that operation fails with `retriever.<kind>.unsupported_operation` before any request is sent.
- `required`: `true` on a reply field fails the read with `unexpected_response` when the reply holds a missing, null,
  or empty value for it, and no partial record is returned. A missing optional field still maps to null.
- `format`: one closed name, `id`, `uuid`, `share-id`, or `strings`, checked on the value with the same checks the
  catalog kinds use on request parameters. `strings` reads a list of strings instead of one scalar; any other value
  shape fails. A free pattern has no form in the language. The failure names the field and never the value.
- `object`: a reply field may hold one nested record instead of a scalar: `user` with `object` holding at most 32
  named sub-fields, each a pointer with its own optional `required`, `format`, and `convert`. One level only: a
  sub-field with its own `object` is refused, and so is an object field with a scalar keyword of its own. A pointer
  that reaches an object or a list is refused, so a whole object is never copied. The mapper reads declared pointers
  only, and the reply tree is dropped after mapping. A failure names the dotted path, never the value.
- `bool_map`: a reply field may hold one open-key map with closed bool values instead of a scalar or a nested
  record: `notificationSettings` with a `pointer` to the reply object and `bool_map: true` keeps every key the reply
  holds, verbatim, sorted, with each value checked as a flag. An optional `fallback_pointer` reads the second reply
  shape the same way, and an optional `required` fails a missing map. A value that is not a bool fails the read with
  `unexpected_response` naming the dotted path, and the value is never kept; the mapper never drops a key. A bool-map
  field takes no `format`, `convert`, or `object`, and no bool map nests inside an `object`. An empty map answers an
  empty record. A failure names the dotted path, never the value.
- **Threat: an open key set smuggles a value the author meant to withhold.** Control: the one open record form keeps
  flags only, so no nested object, text, or list can ride along; every key is kept verbatim with no per-key pointer
  to misdirect; a wrong-shaped value fails the read instead of mapping; and no other open spelling loads, because the
  schema and the Go rule refuse a star sub-field, `patternProperties`, `additionalProperties`, and any undeclared
  value rule. Proof: `TestBoolMapKeepsEveryKeySorted` (every key kept, sorted, with exact flags),
  `TestBoolMapEmptyMapOk` (an empty map answers an empty record),
  `TestBoolMapNonBoolFailsNamingPath` (a non-bool fails naming the dotted path),
  `TestGatewayServesNotificationWritesOnTheEngine` (both notification writes run on the engine with every row
  field), `TestBoolMapRefusesScalarKeywords`, and `TestDynamicRecordFieldsStayRefused`.
- `markdown`: the output field that holds the Markdown body, for a document read. It must name a field.
- `document`: the Markdown layout of a read of one object, rendered from the mapped fields into the `markdown`
  output field, in place of a `markdown` key naming a field. It names `title` (the output field whose value renders
  as `# <title>`, one collapsed line), `lines` (one to 32 metadata lines, each `- <label>: <value>` with a constant
  label of 1 to 32 characters), and `body` (the description body). A line names one closed kind: `field` escapes
  its value with the moved `escapeText` algorithm, `verbatim` writes a shape-checked key or date as is, and `pair`
  renders `name (key)` escaped, or the bare key when the name is empty. `join` renders a string list with the fixed
  separator `", "`. `default` fills an empty value with its closed constant; `skip_empty` omits the line when the
  value is still empty; otherwise the line is emitted even when empty, as the Go code emits Reporter. `valid`
  checks the value against the issue-key shape, emptying it when it does not pass. The body names `field` (the
  output field whose converted value it renders), `convert` (`markdown-from-adf`, matching the converter of its
  field), `heading` (the fixed heading of `## <heading>`), and `empty` (the closed constant of a blank
  description). Only a read of one object declares one, and an example may expect `markdown` exactly then.
  The title, every line reference, and the body field name flat output-field names: keys of the operation's own
  `fields` map. A dotted reply path such as `project.name` is not a field name; the loader refuses it, so a layout
  that needs a reply value maps the value to a flat field first and names that field.
  Proof: `TestDocLayoutPlantStaysText` (a planted heading and link stay literal),
  `TestDocLayoutSkipsEmptyLines` (an empty resolution, parent, and due leave no line), and
  `TestDocLayoutDefaultsExact` (`Unassigned` and `(no description)` byte-identical).
- `errors`: a map from HTTP status, from 100 to 599, to a reason from the closed list in the error section.
- `unavailable_message`: the exact body message that proves the method is missing. A 404 carrying exactly this message
  fails as `operation_unavailable` instead of the mapped reason, so one status tells a missing object from a missing
  method. The match is equality with the declared constant only: no pattern, and no substring match on arbitrary text.
  The message never reaches an error: the failure names the method, never the body. The 404 body of any other
  operation is never read. A 404 with any other body falls through to the mapped reason, so a missing object keeps its
  own failure.
- **Threat: a body rule leaks the reply or widens the language.** Control: the rule fires on 404 only, the outcome is
  fixed to `operation_unavailable`, and only the exact declared message matches. Proof:
  `TestUnavailableMessageDistinguishesAMissingMethod` (the router's catch-all fails as unavailable),
  `TestUnavailableMessageKeepsAMissingObjectNotFound` (any other 404 keeps the mapped reason), and
  `TestUnavailableMessageLeavesOtherStatusesAlone` (a 401 with the same body keeps its own error).

The review text lists each operation with required or checked fields in its own block, with each required field and
each checked field with its closed format name. The review text lists each operation with a nested record field in
its own block, with the object field and each sub-field. The review text lists each operation with an open-key map with closed bool values in its own block, with the map
field and its pointer. The review text lists each first-item read in its own
block, with the operation and the list pointer as `first of /data`. The review text lists each merge read in its own
block, with the label field and each list as `<label> of <pointer>`, and it joins the required, checked, and nested
fields of every list under the same operation. The bool-map block covers the map fields of every list too. The review text lists each tree read in its own block, with the items
pointer, the children pointer, the computed fields, the root parent parameter when the read names one, and the depth
cap. The review text lists each fallback read in its own block, with the target operation, the trigger, and the fill
fields. The review text lists each `page-all` read with its fixed page size in place of parameters. The review text
lists each read with an issue key in its own block, with the path parameter, and each read with the page-end rule in
its own block, with the last flag, the cut parameter, and the truncated pointers. The review text lists each read
with a key echo in its own block, with the parameter and the pointer, and each read with built IDs in its own block,
with each field, the ID it builds, and the pointer it builds it from, joining the built fields of every list under
the same operation. The review text lists each read with a document layout in its own block, with the title field,
the line count, and the body field with its heading. The review text lists each operation with wire aliases in its own
block, with each parameter and the wire key it travels under. Dual pointers,
the body-matched 404 rule, and converters add no review block: the review names rules, never reply shapes.

There are no expressions. A pointer reads, and nothing computes.

## Bundled kinds and the ID mapping

After the ports, `outline`, `kiwi`, and `jira` are bundled specs. The loader refuses a built-in kind in the user
folder, and a spec id such as `outline.documents.get` differs from the catalog id `documents.get`, which the CLI and
MCP names come from. The spec module has two entry points: `Parse` serves every file of the user folder, and it
refuses a built-in kind and the `catalog_id` field. `ParseBundled` accepts both, and only the loader of the embedded
specs calls it. The embedded bytes come from `go:embed`, so no file, flag, or catalog field reaches the exception.

A bundled operation may carry `catalog_id: documents.get`. The catalog row stays the source of the CLI name, the MCP
tool name, the flags, the descriptions, the input schema, and the hints. The spec supplies only the request and the
reply rules. A table in the app was rejected, because it would be a second place to drift.

Each `catalog_id` names one existing row of the same built-in kind, once. An operation that no row names, such as a
fallback target, has no `catalog_id` and sets `cli: false` and `mcp: false`. A spec may not claim a row that another
spec of the kind already claims: one file holds each bundled kind, and a repeated claim inside one file never leaves
the validation. The parameters of a claimed operation match the row in names, types, and required flags. The claimed
operations of each kind cover every read row of the kind; a change row may stay unclaimed until its port, because the
app routes it to a gateway meanwhile.

The review text lists each bundled operation in its own block, with the catalog row it claims, or with no catalog row
and its surfaces off.

## Examples

Each example holds `request` (parameter values), `response` (status, optional constant headers, and body), and `expect`
(the output fields that the reply must map to). The `request` keys must name parameters of the operation, and the
`expect` keys must name its output fields. A reply may not carry an `Authorization` header, and no string of an example
may hold a bearer or basic credential or a token-shaped run of 32 characters. A spec holds no host, so an example names
none. The bundled fixture uses `example.test` names only.

## Limits for the Outline port

The bundled `outline.yml` serves the 66 catalog reads with a `catalog_id` and an example each, the fallback target
`documents.export` with both surfaces off, and all 99 write, destructive, and admin rows with a `catalog_id` and an
example each. The Outline gateway runs each operation on the engine, with the not-found translation the retired Go
gateway mapped. The earlier language limits no longer block a read or a write: `POST` with `read_over_post`
([POST reads](#post-reads)), paging in the body (`in: body`) and the `page-all` rule for an operation with no page
parameters ([Pagination](#pagination)), required and format-checked fields including string lists, nested records,
the first item, merged lists, trees, the fallback, dual reply pointers with a body-matched 404 rule, the `hostname`
converter, and the `text-from-editor-data` converter ([Response](#response)).

All 66 reads run on the engine: `collections.list` pages with the `page-all` rule under the `list-collections` role,
failing at the cap with the page-cap error, `documents.get` maps its wrapped and unwrapped shapes through fallback
pointers, telling a missing document from a missing method by the exact 404 body message, and the comment reads map
editor data through the converter.

All 99 writes run on the engine, including the seven whose parameters need the `Icon`, `URLList`, or
`EventList` kinds, and the two notification subscription writes, whose `notificationSettings` reply field is an
open-key map with closed bool values: the engine keeps every event type the wiki returns, sorted, with each value
checked as a flag. A value that is not a bool fails the read with `unexpected_response` naming the dotted path
instead of being dropped, which is stricter than the Go projection and accepted. No other open record form exists:
the spec still refuses a star sub-field, `patternProperties`, `additionalProperties`, and any undeclared value rule,
so the nested object with explicitly named sub-fields and the bool map stay the only record mechanisms.

A spec named `outline` is also refused in the user folder, because the kind is a built-in source. A bundled Outline
spec takes the exception for embedded files only, as described in
[Bundled kinds and the ID mapping](#bundled-kinds-and-the-id-mapping).

## Limits for the Jira port

The bundled `jira.yml` serves the five catalog reads with a `catalog_id` and an example each: `jira.whoami`,
`jira.projects.list`, `jira.issues.search`, `jira.issues.get`, and `jira.comments.list`. The Jira gateway runs
each operation on the engine, with the document-not-found translation the Go gateway maps: the engine
reports every 404 as a plain not found, and the gateway restores the document sentinel for the two reads whose path
holds the issue key. The spec measures 7 operations in 14,946 bytes, 6 percent of the 256 KiB cap. A spec named
`jira` is refused in the user folder, because the kind is a built-in source, as with `outline`.

`whoami` ports with no new feature: three required and optional fields over `GET /rest/api/3/myself`. The
collection walk is an offset read of the project search with the page-end rule (`last_pointer` on `/isLast` and
`cut_to_limit` on the page size), walking at most twenty pages: the engine sends the Jira `maxResults` and `startAt`
query keys, which the `page-all` rule cannot spell, because it sends only the fixed `limit` and `offset` keys. The
tree walk is a second cursor read of the search endpoint with the page-end rule and the computed `truncated` flag,
walking at most five pages: the gateway passes the JQL of one project and maps the flat issues to parent and depth
itself. The single-page search is a cursor read with `max_pages: 1` for the same reason: only a cursor rule may
declare the `truncated` block, and the page token never travels on its one page. Its `cursor_param` names the
`limit` parameter, because the rule must name a declared body parameter and the catalog row declares no token
parameter; the engine never sends a cursor on one page, so the name is validation ballast only. The project list
and the comment
list send the Jira wire names through the `query_key` alias: `limit` rides as `maxResults` and `offset` rides as
`startAt`, as the Go gateway sends, so the two read scenarios that pin the Jira names pass unchanged. The document
layout of `issues.get` names flat output-field names only, as the
[response](#response) rule requires, and the intermediate fields stay in the record beside the five catalog fields.

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
that the Outline, Kiwi, and Jira connectors can run as specs. All nine of them, POST reads (E1), body paging
(E2), required and validated fields (E3), nested records with an allow-list (E4), the first item or not found
(E5), merged lists with constant labels (E6), computed tree fields (E7), the fallback read (E8), and the bundled
kinds with the ID mapping (E9), are built and described above. The Outline port then needed six more, built by pull
request 17 and described above with it: the `Icon`, `URLList`, and `EventList` parameter kinds, the `page-all` rule,
dual reply pointers with the body-matched 404 rule, the `text-from-editor-data` converter, and the open-key map with
closed bool values that serves the notification subscription replies. This page documents only what the module accepts
today, and each extension moves here when its pull request lands. Pull request 18 then served the `markdown-from-adf`
converter, described above with the response converters. The Jira port then needed six more still: pull request 33
built the first two, the `IssueKey` parameter kind and the page-end rule, described above with the parameters, the
pagination, and the response. Pull request 34 built the next one, the `jira-time`, `jira-date`, and `jira-title`
text converters, described above with the response converters. Pull request 35 built the next two, the key echo and
the built IDs, described above with the response and its review blocks. Pull request 36 built the last one, the
document layout with its title, closed line kinds, and description body, described above with the response and its
review block. Pull request 37 built one more still, the query key: an `in: query` parameter may ride under a wire
alias such as `maxResults` and `startAt`, described above with the parameters and its review block. All seven Jira
extensions are built: the issue key, the page end, the text converters, the key echo,
the built IDs, the document layout, and the query key.
