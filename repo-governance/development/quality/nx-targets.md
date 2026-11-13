---
description: >-
  Sets the Nx target names every project exposes and what each target must do, so that a gate can run one name across
  all projects with the same meaning.
when_to_use: >-
  Use when adding an Nx project or target, declaring a cache input, or when a gate calls a target by name.
---

# Nx Targets

The workspace uses Nx to run tasks across projects. Every project exposes the same target names with the same meaning,
so a gate can run one name across all of them. No project exists yet. The first project adopts these names when it is
added.

## Required Targets

| Target             | Meaning                                                     |
| ------------------ | ----------------------------------------------------------- |
| `lint`             | Static checks. Read-only.                                   |
| `test:unit`        | Unit tests.                                                 |
| `test:integration` | Integration tests.                                          |
| `test:e2e`         | End-to-end tests against the built binary.                  |
| `build`            | Produces the binary or the library output.                  |
| `format`           | Rewrites files to the formatter's output. Mutates the tree. |
| `format:check`     | Fails when the formatter would change a file. Read-only.    |

## Rules

- A target's name says what it does, and a target does one job. A combined job gets a combined name.
- A target declares every input it reads and every output it writes, so that the cache is correct.
- Only `format` writes to the source tree. It runs in a mutation gate, never in a check gate.
- A target that needs the network is marked as such and stays out of the cached path that runs before merge.
- The local Nx cache is regenerable. Keep unique work and secrets out of it, as the [external Nx cache
  eviction](../practice/resource-aware-development/001-external-nx-cache-eviction.md) rule states.
- Compute runs through HIPPO, as the [resource-aware development](../practice/resource-aware-development.md) standard
  requires.
