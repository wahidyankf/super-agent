---
title: Module layout for connectors and profiles
description: Which new Go module owns each part of the spec engine, and what each module may import.
category: explanation
tags:
  - explanation
  - architecture
  - go
  - planned
---

# Module layout for connectors and profiles

This page extends [library layering](../../library-layering.md) for
[ADR 001](../001-user-defined-connectors-and-profiles.md).
It changes no code. The one-`go.mod`-per-module rule holds: each new part is its own library under `libs/`, named
`retriever-<part>`, with its own Nx project and tags.

## New modules

| Module                  | Type tag           | Owns                                                           |
| ----------------------- | ------------------ | -------------------------------------------------------------- |
| `retriever-spec`        | `type:spec`        | The spec model, the JSON Schema file, validation, review text. |
| `retriever-engine`      | `type:engine`      | Rendering, safeguards, mapping, paging, the replay stub.       |
| `retriever-plugin`      | `type:plugin`      | Child start, the pin check, JSON-RPC 2.0 framing, env scrub.   |
| `retriever-conformance` | `type:conformance` | The conformance suite and its report.                          |

The replay stub is the package `engine/replay`. It answers requests from canned exchanges, in memory.

## Who may import whom

| Module                  | Production code may import             | Test code may also import |
| ----------------------- | -------------------------------------- | ------------------------- |
| `retriever-spec`        | `retriever-contract`                   | `retriever-testkit`       |
| `retriever-engine`      | `retriever-contract`, `retriever-spec` | `retriever-testkit`       |
| `retriever-plugin`      | `retriever-contract`                   | `retriever-testkit`       |
| `retriever-conformance` | contract, spec, and engine             | `retriever-testkit`       |

```text
  retriever-cli (app) --> runtime, spec, engine, plugin, conformance, contract
  retriever-conformance --> engine --> spec --> contract
  retriever-plugin --------------------------> contract
  retriever-runtime -------------------------> contract
```

Each arrow points from the importer to the module it imports. The app also names the testkit in its tests.

## Rules in words

- All existing rules hold. Nothing imports the app. Production code never imports the testkit.
- The spec module is pure. Its production code imports none of the standard packages that reach outside the process,
  the same list the contract follows. It takes bytes and returns a model, so the app reads the files.
- The engine imports `net/http` and the TLS packages, as the connectors do. It imports no `os`, `os/exec`,
  `path/filepath`, `io/fs`, or `syscall`. Time and sleep come through seams.
- Only the plugin module imports `os/exec`. The architecture test fails on any other production import of it. This
  keeps the plugin the only code-execution path.
- The engine and the plugin do not import each other. Both implement the ports `SourceGateway`, `SourceReader`, and
  `SourceWriter`, so the runtime sees one shape.
- The conformance module is production code, because the app exposes it. It imports no testkit and no outside-world
  package. It drives the engine over the replay stub, which runs in memory.
- The runtime imports neither spec nor engine. The app reads spec bytes through the runtime's file-system port, builds
  the model with the spec module, and hands the runtime the set of known kinds and one builder for each.
- A ported Go connector module leaves the dependency graph in the pull request that retires it.

## Changes to existing modules

- **Contract.** The kind set opens: a loaded spec registers a kind. Row registration accepts rows built from a spec.
  This is a minor version bump.
- **Runtime.** The config paths learn the profile folder and the spec folder. The catalog loader takes the set of known
  kinds as a parameter, and it enforces the profile prefix on credential names.
- **Testkit.** It gains a loopback HTTPS form of the generic stub, driven by the same exchanges as the replay stub, for
  tests under the `integration` tag.
- **App.** It embeds the bundled specs from `examples/connectors/`, adds `--profile` and `--connectors-dir`, adds
  `connector validate` and `connector test`, and builds the CLI and MCP surfaces from the loaded specs. The composition
  root keeps all real file, process, and network access.

## Third-party packages

The spec module owns the YAML and JSON Schema libraries. No other module imports them.

## As built

The loopback stub for spec kinds does not live in the testkit. The testkit may import only the contract, and the stub
needs the engine's replay stub and the spec module. So the stub lives in `apps/retriever-cli/tests/specstub`, under the
`integration` tag. Only the composition-root tests in `cmd/retriever` and the integration-tagged part of the behaviour
suite import it. The testkit has no spec stub.
