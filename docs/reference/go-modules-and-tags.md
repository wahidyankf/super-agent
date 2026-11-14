---
title: Go modules and tags
description: >-
  Module paths, Nx projects and tags, requirements, versions, and the tag format of the app and the eight libraries.
category: reference
tags:
  - reference
  - go
  - modules
  - tags
  - versions
---

# Go modules and tags

Nine Go modules sit in this repository: the app and eight libraries. Each module has its own `go.mod`. No
`go.work` file joins them. This page states the facts. The rules for imports are in
[library layering](../explanation/library-layering.md).

## Modules

Every module path is `github.com/wahidyankf/super-agent/` followed by the module's directory. Every module declares
`go 1.27.0` and `toolchain go1.27.4`.

| Directory                          | Nx project                    | Type tag         | Source tag       |
| ---------------------------------- | ----------------------------- | ---------------- | ---------------- |
| `apps/retriever-cli`               | `retriever-cli`               | `type:app`       | none             |
| `libs/retriever-contract`          | `retriever-contract`          | `type:contract`  | none             |
| `libs/retriever-runtime`           | `retriever-runtime`           | `type:runtime`   | none             |
| `libs/retriever-testkit`           | `retriever-testkit`           | `type:testkit`   | none             |
| `libs/retriever-connector-outline` | `retriever-connector-outline` | `type:connector` | `source:outline` |
| `libs/retriever-connector-kiwi`    | `retriever-connector-kiwi`    | `type:connector` | `source:kiwi`    |
| `libs/retriever-connector-jira`    | `retriever-connector-jira`    | `type:connector` | `source:jira`    |
| `libs/retriever-spec`              | `retriever-spec`              | `type:spec`      | none             |
| `libs/retriever-engine`            | `retriever-engine`            | `type:engine`    | none             |

Every Nx project also carries `lang:go` and `domain:retriever`. The app also carries `platform:cli`. The type tag names
the module's kind, and the architecture test reads it. The `lang:go` tag makes the continuous-integration Go job run for
an affected project.

## Requirements

Each module names its sibling modules in its `go.mod`, at `v0.4.0`. Each requirement has a `replace` directive to the
sibling's folder, so a build in this repository uses the working tree.

- `apps/retriever-cli` requires the libraries that it builds on, including `libs/retriever-spec` and
  `libs/retriever-engine`, which load and run connector specs. Its Nx `implicitDependencies` lists the same libraries.
- `libs/retriever-contract` requires no sibling.
- `libs/retriever-spec` requires `retriever-contract` alone.
- `libs/retriever-engine` requires `retriever-contract` and `retriever-spec`.
- `libs/retriever-runtime` and `libs/retriever-testkit` each require `retriever-contract`.
- Each connector requires `retriever-contract` and `retriever-testkit`. It requires the testkit for its tests only.
  Its production code imports the contract alone.

The Nx `implicitDependencies` of each library name the siblings that its Go files import, tests included. The
architecture test checks that each list is complete.

## Versions

- Each library is at `v0.4.0`. The eight libraries are released in lockstep, so one version applies to all of them.
  `libs/retriever-spec` and `libs/retriever-engine` have no tag yet, and they take the lockstep version when the
  libraries are next released.
- `ContractVersion` in [contract.go](../../libs/retriever-contract/contract.go) is `0.4.0`. Its comment says that the
  value matches the release tag of the libraries.
- The app module has no version of its own. The app is released with the repository tag, and the latest tag is
  `v0.2.0`.

A contract change bumps the minor version in v0.x, because the project is alpha. The [changelog](../../CHANGELOG.md)
lists each change under the release that makes it.

## Tags

| Module  | Tag form             | Example                          |
| ------- | -------------------- | -------------------------------- |
| app     | `vX.Y.Z`             | `v0.2.0`                         |
| library | `libs/<name>/vX.Y.Z` | `libs/retriever-contract/v0.4.0` |

The library tag has the module's folder as its prefix. Go reads a tag with a path prefix as a version of the module in
that folder, so the prefix must match the folder exactly.

- Create each tag as an annotated tag, as the app tags are.
- Tag the contract first. Tag each library that requires it after that, so every requirement names a tag that exists.
- A consumer outside this repository resolves a requirement from the tags, and it ignores the `replace` directives of
  its dependencies. A library tag must exist before an outside build can use it.
- A tag never moves. A version that has a tag is spent, so the next change takes the next version.

The tags that exist today are `v0.1.0` and `v0.2.0`. No `libs/` tag exists yet. The
[cut an alpha pre-release](../how-to/cut-an-alpha-pre-release.md) guide covers the app tag only.
