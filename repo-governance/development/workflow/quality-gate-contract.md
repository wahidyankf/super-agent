---
description: >-
  Defines how quality gates are declared, which surfaces run them, what they report, and the rule that a gate which
  cannot run fails its surface.
when_to_use: >-
  Use when adding, changing, or removing a gate, or when a gate result is unclear.
---

# Quality Gate Contract

A quality gate is one check that runs at one or more surfaces. Gates are declared in `repo-config.yml`, and RHINO runs
them. HIPPO admits the local compute that they need.

## Declaration

- Each gate has an identifier, a type (`check` or `mutation`), a direct executable with typed arguments, and the
  surfaces that run it.
- A gate names its inputs. A gate that reads staged files binds to the git index. A gate that reads a commit range binds
  to that range.
- A gate declares no default for a missing value. A missing input fails the gate.

## Surfaces

- **pre-commit:** the staged files, before each commit.
- **commit-msg:** the commit message, checked for the public-safety rules.
- **pre-push:** the pushed range, before a push.
- **pull-request:** the whole pull request, in CI and locally before merge.

## Results

- A check reports `0` when it is clean and a finding when it fails.
- A check that cannot run fails its surface. It never reports a pass.
- A mutation applies its fix locally, and CI verifies that the tree is clean. A mutation never runs unseen.

## Quick and Full Gates

- The quick gate runs at pre-commit. It covers formatting, governance checks, and the public-safety checks on the staged
  tree.
- The full gate runs on the pull request. It covers every check, the leak scan over the diff, the conformance runner for
  the command-line tool, and the tests at each level.
- The conformance runner runs before merge, not after release. A contract break therefore cannot land unnoticed. See the
  [tiers record](../../conventions/command-line-interface/009-tiers-here.md).

## Running Gates

Run the gates from the worktree through HIPPO, for example:

```text
./hippo run --class ephemeral --resource-tier standard --disk-path . -- ./rhino gate run --surface pull-request
```

## Changing a Gate

A gate change is a rule change, so it follows [Rules Propagation](../../workflows/quality/rules-propagation.md). A gate
is never weakened to make a change pass. See [Fail Closed](../../principles/fail-closed.md).
