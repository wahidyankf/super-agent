---
title: "ADR 001: User-defined connectors and profiles"
description: Decides on declarative connector specs, one engine, an opt-in plugin, and separate profiles.
category: explanation
tags:
  - explanation
  - decision
  - planned
---

# ADR 001: User-defined connectors and profiles

**Status:** accepted on 2026-11-14. Every change below is planned, and none is built.

## Context

A connector is Go code today. [Add a connector](../../how-to/add-a-connector.md) lists twelve steps, so a user who needs
another service must change this repository.

One catalog can also hold every context, so one run can reach all their credentials.

## Decision

- **D1. retriever stays pure.** It never calls an LLM and holds no clever logic. It does not guard actions on a source.
  The role-based access of the API token is the guard ([who guards what](../who-guards-what.md)).
- **D2. A connector is data.** A YAML spec with `version: 1`, checked by a JSON Schema, declares the kind, the endpoint
  rule, the authentication by variable name only, the operations, the CLI and MCP flags, and examples per operation.
  [Connector spec fields](../../reference/connector-spec-fields.md) drafts it.
- **D3. One generic engine runs every spec.** It applies the safeguards of the built-in connectors to all of them: HTTPS
  and host-only endpoints, a key bound to its own host, no redirects, size and time limits, redaction, stable error
  codes `retriever.<area>.<reason>`, the exit-status contract, and no retry of a write with an unknown outcome. A write
  simply runs: release `v0.2.0` removed `--dry-run`, `--yes`, and `--confirm` on purpose, and the engine adds none.
  The `access` of an operation (read, write, admin) labels MCP hints and is no ceiling.
- **D4. A plugin is the escape hatch.** A catalog source may name an out-of-process plugin that speaks JSON-RPC 2.0 over
  standard input and output. It is off by default and opt-in for each source. It needs a path and a SHA-256 pin, and it
  runs with a scrubbed environment and only its own credential.
- **D5. Specs live outside the repository,** in a folder under the user configuration folder, or the one a flag names.
  Bundled specs for Outline, Kiwi, and Jira stay in the repository with neutral example hosts. The schema suits channel
  connectors that send, but none ships in the first release.
- **D6. A connector costs a spec and fixtures.** BDD covers the engine once, against one generic stub. A conformance
  suite runs against any spec, through `connector validate <spec>` and `connector test <spec>`. Unit tests stay
  hermetic. Real I/O lives in the composition root under the `integration` tag.
- **D7. Profiles separate contexts.** Each profile has its own catalog. Exactly one profile is active per run, chosen by
  flag or environment variable. No run reads or merges two profiles, and no credential is shared between them.

## Options considered

- **Go code per connector.** Rejected. Every service costs a module and a release, and a user cannot add one.
- **A scripting or template language in specs.** Rejected. It makes untrusted input a program that no suite can bound.
  Specs get typed slots only.
- **WebAssembly plugins.** Rejected for now. The sandbox is strong, but it adds a runtime, an author toolchain, and a
  host interface to own. Reopen it if a sandbox becomes necessary.
- **Embedding an LLM.** Rejected. It breaks D1 and makes scripted contracts non-deterministic.

## Consequences

- A new service costs a spec and fixtures, not Go tests.
- A loaded spec supplies a kind, so the contract's kind set opens. That is a minor version bump.
- retriever does not stop a write that the token allows. The engine cannot check that `access` is true, so
  `connector validate` prints each operation for review, and the token's own access guards the source.
- The plugin host starts the first child process of the tool. A failed start exits `2`.
- Kiwi resolves names to IDs, and Jira converts a document tree to Markdown. The spec offers declared lookups and closed
  converters. If they fall short for Kiwi, it stays Go code until schema version 2.
- The [threat model](./001-user-defined-connectors-and-profiles/001-threat-model.md) names each control and its proof.

## Out of scope

The [software-factory](../software-factory-note.md) orchestration, channel connectors, and the `vault` kind.

## Migration order

Engine, then Outline, Kiwi, and Jira ported onto it, then the conformance suite, the plugin host, and profiles. A Go
connector retires in its port, once its scenarios pass unchanged. The
[module layout](./001-user-defined-connectors-and-profiles/005-module-layout.md) says which module owns which part. The
[pull-request split](./001-user-defined-connectors-and-profiles/006-pull-requests-engine.md) gives twelve steps.
