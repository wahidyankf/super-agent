---
description: >-
  Explains how the repository applies its stack packs and gates: which stack owns each file, how Nx projects and the
  HIPPO and RHINO gates reach them, and how a new stack is added.
when_to_use: >-
  Use when adding a stack, a project, or a gate, or when a file has no clear owning stack.
---

# Repository Adapter

The repository adapter ties the stack standards to the repository's projects and gates. It answers three questions:
which stack owns a file, which project runs its checks, and which gate reports them.

## Ownership

Each path belongs to one stack.

- `apps/` and `libs/` hold Go code, one Go module per folder. A module's own `scripts/` folder, which holds its
  exclusions file, belongs to that module's stack.
- The root `scripts/` and `.husky/` hold shell code.
- `repo-governance/`, `docs/`, `specs/`, and `agents/` hold markdown and specifications.

A file with no owner is a defect, and the repository check reports it.

## Projects and Targets

- The repository has eight Go modules: the app `apps/retriever-cli` and seven libraries under `libs/`. Each has its own
  `go.mod` and is one Nx project in its folder.
- Each Nx project declares its targets in its project configuration, using the [Nx targets standard](../nx-targets.md).
  A Go target runs from its module's folder, so it reads that module's `go.mod`.
- `repo-config.yml` declares each project under `extensions.software-development.projects`.

## Gates

- HIPPO admits local compute, and RHINO runs the gates.
- Gates are declared in `repo-config.yml` with a direct executable and typed arguments. Each one names the surfaces that
  run it.
- Each gate is either a check or a mutation. A mutation applies its fix locally, and CI then verifies that the tree is
  clean. See the [quality gate contract](../../workflow/quality-gate-contract.md).
- The Go unit-coverage gate is not a repo-config gate. Each module's `test:unit` target runs
  `scripts/go-coverage-gate.sh` from the module's folder. Each module is held to the 99% [coverage
  floor](../testing/meaningful-coverage.md) on its own: one instrumented run of its untagged tests, with no profile that
  merges two modules and no `-coverpkg` across a module boundary. The gate excludes only the `generated` entries of the
  module's `scripts/coverage-exclusions.tsv`. A developer runs these targets through Nx; the pull-request workflow runs
  only the app's targets today.
- The pull-request quality gate starts the Go job for each affected Nx project tagged `lang:go`, and the workflow lists
  the targets that job runs.

## Adding a Stack

1. Write the stack standard and link it from the stacks index.
2. Add the stack's extension entry and projects to `repo-config.yml`.
3. Add the gates the stack needs, each with its surfaces.
4. Add the stack to the table in [Stack Packs](../../../conventions/structure/stack-packs.md).
