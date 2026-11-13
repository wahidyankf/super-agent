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

- `apps/` and `libs/` hold Go code.
- `scripts/` and `.husky/` hold shell code.
- `repo-governance/`, `docs/`, `specs/`, and `agents/` hold markdown and specifications.

A file with no owner is a defect, and the repository check reports it.

## Projects and Targets

- Each Nx project declares its targets in its project configuration, using the [Nx targets standard](../nx-targets.md).
- `repo-config.yml` declares each project under `extensions.software-development.projects`.

## Gates

- HIPPO admits local compute, and RHINO runs the gates.
- Gates are declared in `repo-config.yml` with a direct executable and typed arguments. Each one names the surfaces that
  run it.
- Each gate is either a check or a mutation. A mutation applies its fix locally, and CI then verifies that the tree is
  clean. See the [quality gate contract](../../workflow/quality-gate-contract.md).

## Adding a Stack

1. Write the stack standard and link it from the stacks index.
2. Add the stack's extension entry and projects to `repo-config.yml`.
3. Add the gates the stack needs, each with its surfaces.
4. Add the stack to the table in [Stack Packs](../../../conventions/structure/stack-packs.md).
