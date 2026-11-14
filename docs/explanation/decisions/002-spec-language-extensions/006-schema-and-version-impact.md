---
title: Schema and version impact of the spec language extensions
description: Why the nine extensions are additive to schema version 1, and which module versions change.
category: explanation
tags:
  - explanation
  - versioning
  - planned
---

# Schema and version impact of the spec language extensions

This page belongs to [ADR 002](../002-spec-language-extensions.md). Everything here is planned.

## The schema stays at version 1

All nine features are additive. Each is a new optional field or a new allowed value, and no existing field changes its
meaning:

| Feature | New in the schema                                                      |
| ------- | ---------------------------------------------------------------------- |
| E1      | `read_over_post` on an operation, and `POST` allowed for a marked read |
| E2      | `in` on `pagination`, with the value `body`                            |
| E3      | `required` and `format` on a reply field                               |
| E4      | `object` on a reply field                                              |
| E5, E6  | `first` and `merge` on `response`                                      |
| E7      | `tree` on `response`                                                   |
| E8      | `fallback` on an operation                                             |
| E9      | `catalog_id` on an operation, valid only for a bundled spec            |

Two tests hold this. `TestExistingSpecsStillValidate` runs every spec of the current corpus, including
`example-wiki.yml`, and expects the same result as before. `TestSchemaKeywordLock` fails when the schema holds a keyword
that this ADR does not name (threat L11).

An older reader fails closed. It refuses an unknown field with `retriever.connector.spec_invalid` and the field path,
so a new spec never runs with a feature that the reader lacks. The schema file keeps its name,
`connector-spec-v1.schema.json`.

## Why not version 2

A version 2 would split the corpus for no gain, because the unknown-field rule already protects an older reader. Version
2 stays free for a real break, such as a Kiwi form that declared lookups cannot express (decision D9).

## Module versions

No library tag is cut yet, so these are the versions the `go.mod` files and the changelog declare.

| Module               | Now      | After                                   | Reason                                   |
| -------------------- | -------- | --------------------------------------- | ---------------------------------------- |
| `retriever-spec`     | `v0.4.0` | `v0.5.0`, bumped once in pull request 7 | New optional fields: a minor bump.       |
| `retriever-engine`   | `v0.4.0` | `v0.5.0`, bumped once in pull request 7 | New behaviour behind new fields.         |
| `retriever-contract` | `v0.4.0` | `v0.4.0`, or `v0.5.0` if needed         | Changes only if a record cannot hold E4. |
| Go connector modules | `v0.4.0` | removed                                 | Each leaves the graph in its port.       |

`Field.Value` in the contract record is `any`, so a nested value is possible by type. Pull request 10 first checks that
the renderers, the MCP adapter, and the JSON output handle it. If they do not, the contract moves to `v0.5.0` in that
pull request, and the changelog lists it as **Breaking**.

Removing a Go connector module is also **Breaking** for a Go caller. The changelog entry of each port says so.

## Docs

Each language pull request moves its feature on
[connector spec fields](../../../reference/connector-spec-fields.md) from planned to as built,
in the same change. Until then that page says only that the extensions are planned.
