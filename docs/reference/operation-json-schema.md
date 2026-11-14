---
title: Operation and spec JSON Schema
description: >-
  The JSON Schema of a declarative connector spec, version 1, which exists, and the operation catalog, which has none.
category: reference
tags:
  - reference
  - schema
---

# Operation and spec JSON Schema

**Status: the spec schema exists.** The file
[connector-spec-v1.schema.json](../../libs/retriever-spec/schema/connector-spec-v1.schema.json) is a JSON Schema,
draft 2020-12. It checks the shape of one connector spec, version 1, and the per-value rules of its fields. The spec
module embeds the file and compiles it once, so the module reads no file at run time. The
[connector spec fields](./connector-spec-fields.md) page lists each field and the rules that the schema and the Go
validation share.

## What exists today

- The spec schema, at `libs/retriever-spec/schema/connector-spec-v1.schema.json`. The spec module checks a document
  against it, and `libs/retriever-spec/schema/README.md` explains its parts.
- The operation catalog is Go code. Each row in the [operations](../../libs/retriever-contract/operations/README.md)
  package describes one operation. No JSON file describes an operation.
- The JSON output of the command-line tool is described in prose in the
  [JSON schemas](./retriever-cli-json-schemas.md) reference. That page is not a JSON Schema file.

## What is not in the schema

- The rules that compare two fields of one spec, such as an operation id against its kind, a slot against a parameter,
  or a resolve against an operation. The Go validation of the spec module checks these after the schema passes.
- A JSON Schema for the operations of the catalog. None exists in this repository, and the Go operation rows remain the
  only definition of an operation.
