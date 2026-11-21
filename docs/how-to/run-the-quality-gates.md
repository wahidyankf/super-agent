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

## Bootstrap a checkout

Install the npm packages once, through HIPPO. The Markdown and formatting gates need them:

```sh
./hippo run --class ephemeral --resource-tier standard --disk-path . -- npm install
```

The `prepare` script then installs the Husky Git hooks. Check the install with the hook health check:

```sh
scripts/check-hooks-installed.sh
```

The check prints `Git hooks: installed` and exits `0` when `core.hooksPath` names `.husky/_` and the three hook files
are present and executable. Otherwise it prints one line per failed condition and exits `1`. Run `npm install` again to
repair a failed install. CI runs `npm ci --ignore-scripts`, which installs no hooks, so the check is for a contributor's
checkout only.

## Run the pull-request surface

From the root of your worktree, run every gate that applies to a pull request. Pass the explicit commit range that the
pull request adds. The `--base` value is the commit `origin/main` points to, and `--head` is the commit under review:

```sh
./hippo run --class ephemeral --resource-tier standard --disk-path . -- ./rhino gate run --surface pull-request --base <origin/main sha> --head HEAD
```

The gates that compare commits, such as the commit-history and public-safety range screens, need both values. Without
them the run exits `2`. Exit status `0` means every gate passed. The pull-request surface is the one that a merge
depends on. See [Deliver a change with a pull request](./deliver-a-change-with-a-pull-request.md).

## Run one gate or one check

Run only some gates of a surface with `--entry`, repeated for each id. For example, to run the Markdown lint and the
word budget:

```sh
./hippo run --class ephemeral --resource-tier light --disk-path . -- ./rhino gate run --surface pull-request --base <origin/main sha> --head HEAD --entry markdownlint --entry word-budget
```

Run a single RHINO command the same way. For example, to check the README indexes only:

```sh
./hippo run --class ephemeral --resource-tier light --disk-path . -- ./rhino md readme-index validate
```

Run `./rhino help` to list every command.

## Which hook runs which surface

Git hooks run the gates automatically. Each hook passes its own surface to RHINO. Every other check runs on the
`pull-request` surface, so a commit or a push never waits on it.

| Git hook   | RHINO surface | HIPPO class and tier        |
| ---------- | ------------- | --------------------------- |
| pre-commit | `pre-commit`  | `transactional`, `standard` |
| commit-msg | `commit-msg`  | `ephemeral`, `light`        |
| pre-push   | `pre-push`    | `ephemeral`, `heavy`        |

The gate ids, their surfaces, scopes, and commands are listed in [Quality gates](../reference/quality-gates.md). The
test levels that run in the workflows are in [Retriever test gates](../reference/retriever-cli-test-gates.md).

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
