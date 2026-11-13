---
description: >-
  Routes local compute through one checksum-pinned admission guard with bounded recovery.
when_to_use: >-
  Use when adopting, wiring, or recovering a HIPPO-guarded command.
---

# Resource-Aware Development

HIPPO is an upstream, checksum-pinned wrapper that starts local compute only when shared capacity and pressure permit; a
remote payload's single guard sits on the target host. This standard governs consumers.

This standard implements [Explicit Over Implicit](../../principles/explicit-over-implicit.md), [Root Cause
Orientation](../../principles/root-cause-orientation.md), [Reproducibility](../../principles/reproducibility.md), and
[Fail Closed](../../principles/fail-closed.md).

## One Outer Guard

- Every local build, test, check, generator, service, or dependency install uses exactly one outer guard, except
  `prettier` and `markdownlint-cli2` runs in any mode over any files: bounded and restartable, they need no admission;
  all other compute stays guarded. See [allowed spellings](resource-aware-development/002-unadmitted-tool-runs.md).
- Pin and verify its release checksum—keep implementation, specifications, and releases upstream.
- Never bypass the guard in a repository that uses it.
- Never nest guards. Invoke an already-guarded script directly; otherwise its parent may hold the capacity it awaits.
- Run the guard's status and recovery commands unguarded to avoid admission deadlock.

## Outcomes

- exit `124`: a limit stopped the work, and the reason says which. For storage, free disk space first, then proceed. For
  capacity or pressure, requeue only with a new schema-1 `never-started` receipt. Otherwise recover the payload.
- exit `125`: the guard could not do its job and started nothing. For a protocol mismatch, drain or upgrade every client
  that shares the root, and never retry blindly. Otherwise correct the configuration or the request.
- exit `2`: the invocation is wrong. Read the diagnostic and fix the command.
- exit `126` and `127`: the command cannot be executed, or it is not there. Fix the path or install the tool.
- exit `1`: the work ran and the answer is empty. That is a result, not a failure. Decide on the empty answer.

Two reasons can share one status; read the namespaced reason on stderr and not the number alone. Child codes pass
through: without a new `never-started` receipt, a child's own `124` stays child-owned.

`hippo.lock` pins executable identity, not semantics. Before changing behavior, read the Hippo repository at the commit
in `hippo.lock`, then align rules, Gherkin, and checks with its documented capabilities—never infer capability from
SemVer or copy a release number into governance.

Never resolve an outcome by bypassing HIPPO, changing class for priority, weakening a gate, raising allotted
concurrency, or deleting possibly live state—corrupt state fails closed; follow upstream recovery.

## Workload Classes

| Class         | For                                             |
| ------------- | ----------------------------------------------- |
| restartable   | repeatable builds, tests, and reads             |
| service       | restartable long-running development            |
| transactional | indivisible mutations or tracked-output writers |

The class follows from the work, never from which class is admitted sooner.

Choose each tier by the work's needs, never to be admitted faster: a tier reserves capacity without capping it.
[Resource Tiers](resource-aware-development/003-resource-tiers.md) defines them.

## Parallelism

Independent commands may enqueue together. Serialize only for dependencies, shared output, indivisible changes, or
proven races—not repository boundaries.

## Boundaries

- Dedicated hosted jobs stay unguarded because they have no local contention.
- Evidence holds capacity and process health, never arguments, paths, origins, credentials, contents, or user data.
- Raw evidence rolls for seven days; compacted daily summaries roll for 30 days under byte caps, so shared logging is
  bounded. `history` reads both without exposing command or path data.
- Track privacy-safe `hippo.identity.json`; contained worktrees keep its source and may add `--tag checkout=worktree
--tag plan=<slug>`. Use the shared root; reserve `HIPPO_ROOT` for isolated tests. A worktree without ignored
  `hippo.local.json` uses its worktree-local wrapper and inherits the primary checkout's policy. Run `./hippo status`,
  `./hippo watch`, and `./hippo history`; on a protocol mismatch, status identifies the peers that must drain.
- Test admission and shedding with synthetic state and pressure only.

A hook held back by a deferral has not failed. Reaching that point never bypasses this guard: the hook's compute moves
to the pull-request gate, and the guard's admission, classes, and tiers stay exactly as declared.

## Enforcement

Tests confirm one guard and declared class per compute entry other than those runs, required pinned capabilities, and no
blind retry of a protocol mismatch or receipt-less shed.

## Modules

- [Nx cache eviction](resource-aware-development/001-external-nx-cache-eviction.md)
- [Unadmitted runs](resource-aware-development/002-unadmitted-tool-runs.md)
- [Resource tiers](resource-aware-development/003-resource-tiers.md)
- [Remote dispatch](resource-aware-development/remote-dispatch.md)
