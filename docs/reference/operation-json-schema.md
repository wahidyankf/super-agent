---
title: Operation and spec JSON Schema
description: >-
  PLANNED. The JSON Schema for operations and for declarative connector specs. It arrives with the declarative-spec pull
  request.
category: reference
tags:
  - reference
  - planned
  - schema
---

# Operation and spec JSON Schema (planned)

**Status: PLANNED.** Nothing on this page exists in the repository yet. The JSON Schema for operations and for
declarative connector specs arrives with the declarative-spec pull request. That pull request decides the file paths and
the command that validates them, and this page names them when it lands.

## What exists today

- The operation catalog is Go code. Each row in the [operations](../../libs/retriever-contract/operations/README.md)
  package describes one operation. No JSON file describes an operation.
- No `*.schema.json` file is in the repository.
- The JSON output of the command-line tool is described in prose in the
  [JSON schemas](./retriever-cli-json-schemas.md) reference. That page is not the planned schema.

## What is planned

- A JSON Schema for the operations of the catalog.
- A JSON Schema for the declarative connector spec.

Until the declarative-spec pull request lands, the Go operation rows are the only definition of an operation.
