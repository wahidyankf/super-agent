---
title: "Pull requests 1 to 6: the engine and the first port"
description: The first six pull requests of ADR 001, each with its scope, acceptance scenarios, and extra gates.
category: explanation
tags:
  - explanation
  - delivery
  - planned
---

# Pull requests 1 to 6: the engine and the first port

This page and [pull requests 7 to 12](./007-pull-requests-ports-and-profiles.md) split
[ADR 001](../001-user-defined-connectors-and-profiles.md) into ordered pull requests. Each leaves `main` working. A
code such as S3 points to the [threat model](./001-threat-model.md).

## Standard gates for every pull request

- The pull-request gate surface.
- For each Go module touched: `lint`, `typecheck`, hermetic `test:unit` at the 99 percent floor, and `test:integration`.
- For the app: `test:conformance`, `bdd:check`, and `test:e2e` when behaviour changes.
- A changelog entry, plus the docs and `repo-config.yml` entries of any new module.

## 1. Contract: open the kind set

The contract accepts kinds and rows registered at start-up. The version moves to 0.4.0. Behaviour stays.

- Given only the three built-in kinds, when the example catalog loads, then `sources list` prints the same document.
- Given a registered kind `example-wiki`, when a catalog names it, then it loads. When a catalog names an unregistered
  kind, then it fails and the message lists the registered kinds.

## 2. The spec module

`retriever-spec` holds the model, the schema file, validation, and the review text. The schema page and the
[spec fields draft](../../../reference/connector-spec-fields.md) become as-built. Extra gate: the purity rule of the
[module layout](./005-module-layout.md). Proves S1, S8, S10, S11, C1, and R2.

- Given a valid minimal spec, when it is validated, then there is no finding, and the review lists each operation.
- Given a spec with one fault, such as a template function, an `Authorization` header, a YAML alias, DELETE labelled
  read, or a key in a query, when it is validated, then it fails with `retriever.connector.spec_invalid`, naming the
  field path and no value.

## 3. The engine: reads and safeguards

`retriever-engine` renders requests, authenticates, maps replies, and pages. It adds the replay stub. Review the 256 KiB
spec cap again here. Proves S2, S3 to S7, S9, and R1.

- Given a list operation and the replay stub, when the engine runs it, then the stub receives one request with the
  declared method, path, and header, and the result holds the mapped fields.
- Given a stub that redirects, replies over the cap, nests deeply, resolves to a private address, or stalls, when the
  engine runs a read, then each case ends with its stable error code, exit status `2` or `124`, and no second request.
- Given a value such as `../admin`, or one with a line break, when the engine renders a request, then it refuses before
  any request leaves.

## 4. The engine: writes and lookups

- Given a write and a stub that times out after the request left, when the write runs, then the engine reports an
  unknown outcome and sends no second request.
- Given a write that the stub accepts, when it runs, then the stub receives one request and no confirmation is asked.
- Given a lookup name that matches two entries, when the write runs, then it fails with the unresolved-name error and
  sends no write.

## 5. The app: spec folder and derived surfaces

The app adds `--connectors-dir` and the default folder, accepts spec kinds in the catalog, honours `spec_sha256`, and
builds CLI commands and MCP tools from loaded specs. An example spec joins `examples/`. Docs: configuration,
commands, and the connector how-to. Proves C2.

- Given a spec and a catalog source of its kind, when the command runs against the loopback stub (`integration` tag),
  then it prints the mapped result and exits `0`.
- Given an operation with `cli: false` and `mcp: true`, when the surfaces build, then the CLI has no such command and
  the MCP list has the tool.
- Given a `spec_sha256` that does not match, when the catalog loads, then it fails with
  `retriever.connector.spec_pin_mismatch` and no request goes out.

## 6. Port Outline reads

A bundled `outline.yml` serves the read operations. A composite builder routes each operation to the spec or the Go
gateway until pull request 7.

- Given the unchanged Outline read, redaction, and transport scenarios, when they run against the engine, then every
  one passes without edit.
- Given the MCP read tools before and after, when they are compared, then names and hints match.
