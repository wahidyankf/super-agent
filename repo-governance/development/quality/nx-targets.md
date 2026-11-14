---
description: >-
  Sets the Nx target names that projects expose and what each target must do, so that a gate can run one name across
  all projects with the same meaning.
when_to_use: >-
  Use when adding an Nx project or target, declaring a cache input, or when a gate calls a target by name.
---

# Nx Targets

The workspace uses Nx to run tasks across projects. Every project exposes the same target names with the same meaning,
so a gate can run one name across all of them. The one exception is `test:e2e`, which only a project that builds a
binary declares. The first project, `apps/retriever-cli`, exposes the required names. It also exposes
`test:conformance`, `bdd:check`, and `deps:audit`, which are listed in its
[test gates reference](../../../docs/reference/retriever-cli-test-gates.md).

## Required Targets

| Target             | Meaning                                                     |
| ------------------ | ----------------------------------------------------------- |
| `lint`             | Static checks. Read-only.                                   |
| `test:unit`        | Unit tests.                                                 |
| `test:integration` | Integration tests.                                          |
| `test:e2e`         | End-to-end tests against the built binary. See the note.    |
| `build`            | Produces the binary or the library output.                  |
| `format`           | Rewrites files to the formatter's output. Mutates the tree. |
| `format:check`     | Fails when the formatter would change a file. Read-only.    |

`test:e2e` is required only for a project that builds a binary, which today is `apps/retriever-cli`. A library builds no
binary, so it does not declare `test:e2e`, and no placeholder stands in for it. A library's end-to-end behaviour is
covered by the app's end-to-end suite, which runs it through the built binary.

## Rules

- A target's name says what it does, and a target does one job. A combined job gets a combined name.
- A target declares every input it reads and every output it writes, so that the cache is correct.
- Only `format` writes to the source tree. It runs in a mutation gate, never in a check gate.
- A target that needs the network is marked as such and stays out of the cached path that runs before merge.
- The local Nx cache is regenerable. Keep unique work and secrets out of it, as the [external Nx cache
  eviction](../practice/resource-aware-development/001-external-nx-cache-eviction.md) rule states.
- Compute runs through HIPPO, as the [resource-aware development](../practice/resource-aware-development.md) standard
  requires.
