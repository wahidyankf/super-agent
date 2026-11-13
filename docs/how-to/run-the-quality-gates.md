---
title: Run the quality gates
description: How to run the quality gates through HIPPO and RHINO, which hook runs what, and how to read a failure.
category: how-to
tags:
  - how-to
  - quality
  - gates
---

# Run the quality gates

super-agent runs its checks through two tools:

- **HIPPO** admits each command against the host's free CPU and memory, so several checkouts can work on one machine
  without overloading it. It runs the command under supervision.
- **RHINO** runs the gates declared in `repo-config.yml` and reports every violation it finds.

## Install the dependencies first

The Markdown gate needs the npm packages. Install them once, through HIPPO:

```sh
./hippo run --class ephemeral --resource-tier standard --disk-path . -- npm install
```

The `prepare` script then installs the Husky Git hooks.

## Run the pull-request surface

From the root of your worktree, run every gate that applies to a pull request:

```sh
./hippo run --class ephemeral --resource-tier standard --disk-path . -- ./rhino gate run --surface pull-request
```

Exit status `0` means every gate passed. The pull-request surface is the one that a merge depends on. See [Deliver a
change with a pull request](./deliver-a-change-with-a-pull-request.md).

## Run one check

Run a single RHINO command the same way. For example, to check the README indexes only:

```sh
./hippo run --class ephemeral --resource-tier light --disk-path . -- ./rhino md readme-index validate
```

Run `./rhino help` to list every command.

## Which hook runs which surface

Git hooks run the gates automatically. Each hook passes its own surface to RHINO.

| Git hook   | RHINO surface | HIPPO class and tier        |
| ---------- | ------------- | --------------------------- |
| pre-commit | `pre-commit`  | `transactional`, `standard` |
| commit-msg | `commit-msg`  | `ephemeral`, `light`        |
| pre-push   | `pre-push`    | `ephemeral`, `heavy`        |

The gate ids and the surfaces they run on are listed in [Quality gates](../reference/quality-gates.md).

## When a gate fails

- Read the finding, and fix its cause. Do not disable a gate, loosen its configuration, or add an exception to make
  it pass.
- RHINO exit `1` means a policy found a violation, or a gate child failed. Exit `2` means the invocation, the root,
  the configuration, or the declared boundary was unusable. Fix the command or the configuration.
- RHINO exit `126` means a gate tool could not be executed. Exit `127` means it was not found. Install or fix that
  tool.
- HIPPO exit `124` means a limit stopped the work. Exit `125` means the guard could not start, so nothing ran. Read
  the reason on stderr. Do not change the class or the tier to be admitted sooner. Choose each tier by what the work
  needs.
- A scan that cannot run is a failure. The public-safety gates never treat a scan error as a pass.

## Where the gates are declared

The gates are declared in the `gates` section of `repo-config.yml`. The tool versions are pinned in `hippo.lock` and
`rhino.lock`. To change a pin, follow [Repin HIPPO and RHINO](./repin-hippo-and-rhino.md).
