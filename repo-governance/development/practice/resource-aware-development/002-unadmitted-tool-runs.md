---
description: >-
  Fixes which spellings of `prettier` and `markdownlint-cli2` run without admission, and why every other spelling stays
  guarded.
when_to_use: >-
  Use when running, wiring, or reviewing a guard or hook that lets these two tools skip admission.
---

# Unadmitted Tool Runs

Runs of `prettier` and `markdownlint-cli2`, in any mode over any files, need no admission. Each pass is bounded and
restartable, so queueing it behind heavy runs wastes time and saves no capacity. All other compute stays guarded.

## Allowed Spellings

A boundary hook or guard allows exactly two launcher spellings, where `<tool>` is one of the two tools:

- `npx --no-install <tool>`
- `npm exec --no -- <tool>`, with the `--` optional

A direct call to the installed tool, or a package script that runs only these tools, needs no wrapper either.

## Denied Spellings

- `npx --no <tool>`: when the tool is missing it exits 0 without running, so a `--check` gate passes falsely.
- `npm exec --no-install <tool>`: npm only warns and still installs the missing tool.
- A version suffix such as `prettier@3`, a path, any other tool, any other flag before the tool, and a quoted tool name,
  which fails closed.
- A compound line in which any other segment is guarded compute.

## Enforcement

**Unenforced by decision** for ad hoc runs, because a pass that skips admission harms no shared capacity. The guard and
any native hook allow the two spellings and deny the rest. The guard checks the first word of each segment, so a
launcher such as `xargs` or `find -exec` can hide a guarded verb; that gap predates this rule and stays open here.
