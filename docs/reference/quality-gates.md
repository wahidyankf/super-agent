---
title: Quality gates
description: The gate inventory of super-agent: every gate, its level, its surface, its scope, and its command.
category: reference
tags:
  - reference
  - quality
  - gates
---

# Quality gates

super-agent runs its checks on four surfaces: `pre-commit`, `commit-msg`, `pre-push`, and `pull-request`. The end-to-end
level also runs in a scheduled workflow. The registry in `repo-config.yml` declares each gate and the surfaces that run
it. RHINO runs the gates. The Git hooks call one surface and copy no command. See
[Run the quality gates](../how-to/run-the-quality-gates.md) to run them yourself.

This page is the gate inventory. Each row gives the gate, the surface or workflow, the scope, and the command. A gate on
two surfaces has one row per surface.

## Local hooks

Each local surface holds only the entries that its allowed list names. Every other check runs on the `pull-request`
surface, so a commit or a push never waits on it.

| Hook       | Gate                  | Scope        | Command                                                     |
| ---------- | --------------------- | ------------ | ----------------------------------------------------------- |
| pre-commit | `format-staged`       | staged files | `./rhino gate run --surface pre-commit`                     |
| commit-msg | `commit-message`      | the message  | `./rhino gate run --surface commit-msg --message-file "$1"` |
| pre-push   | `public-safety-range` | pushed range | `./rhino gate run --surface pre-push --push-updates-stdin`  |

- `format-staged` formats the staged files and applies the change to the index, so the commit holds the formatted files.
- The hooks run through HIPPO. Their class and tier are in [Run the quality gates](../how-to/run-the-quality-gates.md).
- `scripts/check-hooks-installed.sh` is a bootstrap check that the three hooks are installed and executable. No workflow
  runs it.

## Registry gates

These gates are declared in the `gates` section of `repo-config.yml`. None of them runs a test.

| Gate                           | Surface      | Scope         | Command                                            |
| ------------------------------ | ------------ | ------------- | -------------------------------------------------- |
| `public-safety-tree`           | pull-request | whole tree    | `bash scripts/public-safety/check.sh`              |
| `public-safety-range`          | pre-push     | pushed range  | `bash scripts/public-safety/check.sh`              |
| `public-safety-range`          | pull-request | commit range  | `bash scripts/public-safety/check.sh`              |
| `public-safety-commit-message` | pull-request | each message  | `./scripts/public-safety/check-commit-message`     |
| `commit-message`               | commit-msg   | the message   | `bash scripts/check-commit-message.sh`             |
| `commit-message`               | pull-request | each message  | `bash scripts/check-commit-message.sh`             |
| `commit-history`               | pull-request | commit range  | `bash scripts/check-commit-range.sh <base> <head>` |
| `format-staged`                | pre-commit   | staged files  | `./scripts/format-staged`                          |
| `format-staged`                | pull-request | changed files | `./scripts/format-staged`                          |
| `remote-dispatch-refusal`      | pull-request | whole tree    | `bash scripts/check-remote-dispatch-refusal.sh`    |
| `repo-config`                  | pull-request | whole tree    | `./rhino repo-config validate`                     |
| `harness-adapters`             | pull-request | whole tree    | `./rhino harness adapters validate`                |
| `governance-vendor`            | pull-request | whole tree    | `./rhino governance vendor validate`               |
| `env-validate`                 | pull-request | whole tree    | `./rhino env validate`                             |
| `convention-emoji`             | pull-request | whole tree    | `./rhino convention emoji validate`                |
| `markdownlint`                 | pull-request | whole tree    | `npx --no markdownlint-cli2 "**/*.md"`             |
| `md-readme-index`              | pull-request | whole tree    | `./rhino md readme-index validate`                 |
| `md-internal-link`             | pull-request | whole tree    | `./rhino md internal-link validate`                |
| `md-heading-hierarchy`         | pull-request | whole tree    | `./rhino md heading-hierarchy validate`            |
| `md-naming`                    | pull-request | whole tree    | `./rhino md naming validate`                       |
| `md-frontmatter`               | pull-request | whole tree    | `./rhino md frontmatter validate`                  |
| `word-budget`                  | pull-request | whole tree    | `./rhino governance word-budget validate`          |
| `shellcheck`                   | pull-request | changed files | `./scripts/lint-shell`                             |
| `actionlint`                   | pull-request | changed files | `./scripts/lint-workflows`                         |
| `check-hooks-installed-tests`  | pull-request | whole tree    | `bash scripts/check-hooks-installed_test.sh`       |

- `format-staged` on `pre-commit` applies the change to the index. On `pull-request`, CI runs the same formatter and
  verifies that the tree is clean.
- `public-safety-range` on `pre-push` screens the commits that the push publishes. On `pull-request` it screens the
  commit range from `--base` to `--head`.
- `commit-history` lints each commit of the range on its own. `commit-message` on `pull-request` runs once for each
  message in the range.
- `shellcheck` and `actionlint` lint the changed shell scripts and workflows.
- `check-hooks-installed-tests` is the self-test of `scripts/check-hooks-installed.sh`. It builds its own fixture
  repositories.

## Workflow and project checks

These checks are Nx targets, not registry gates. The workflows find every project that declares a target and run one job
for it, so a project joins a check by declaring the target. The pull-request jobs are in
`.github/workflows/pr-quality-gate.yml`. The end-to-end job is in `.github/workflows/scheduled-e2e.yml`.

| Check                     | Runs in                       | Command                               | Level       |
| ------------------------- | ----------------------------- | ------------------------------------- | ----------- |
| `lint`                    | `go-static`, every Go project | `nx run <project>:lint`               | static      |
| `typecheck`               | `go-static`, every Go project | `nx run <project>:typecheck`          | static      |
| `deps:audit`              | `go-static`, where declared   | `go tool govulncheck ./...`           | static      |
| `test:unit`               | `go-unit` and `go-unit-macos` | `nx run <project>:test:unit`          | unit        |
| `test:integration`        | `go-integration`              | `nx run <project>:test:integration`   | integration |
| `test:conformance`        | `go-conformance` and macOS    | `nx run <app>:test:conformance`       | integration |
| `test:behaviour:coverage` | `behaviour-coverage`, once    | `./rhino behaviour coverage validate` | static      |
| cross-compile             | `cross-compile`, the app      | four `go build` targets               | build       |
| `test:e2e`                | `scheduled-e2e.yml`           | `nx run <project>:test:e2e`           | end-to-end  |
| `quality-gate`            | aggregate, one required check | passes when required jobs pass        | none        |
| `test:quick`              | no hook and no workflow       | `nx run <project>:test:quick`         | mixed       |
| RHINO identity            | `rhino-consumer-smoke.yml`    | check the pinned release              | smoke       |
| HIPPO identity            | `hippo-consumer-smoke.yml`    | check the pinned release              | smoke       |

- `test:unit` on macOS runs for the app only, and so does `test:conformance` on macOS.
- `test:integration` runs for every Go project that declares it. `retriever-contract` and `retriever-spec` declare none,
  because they have no integration-tagged test. `test:conformance` runs for the app only.
- `test:behaviour:coverage` is declared by every project. The workflow runs the command once for the repository.
- `test:e2e` runs weekly, on manual dispatch, and on each push to `main`. The app's suite also runs on macOS. No
  pull-request job runs it.
- `quality-gate` is the one required status check. A failed job fails it.
- `test:quick` runs `typecheck`, `lint`, `test:unit`, and `test:behaviour:coverage`, in that order. Every project
  declares it. The pull-request template asks the author to run it, and no workflow does.
- The two consumer smoke checks run on pull requests that change their pin, and every day.

## Exempt layers and reasons

Each exemption names a boundary, not a difficulty. The rules are in [Bindings and
Exemptions](../../repo-governance/development/behaviour-driven-development/001-bindings-and-exemptions.md).

- **Unit level.** Never exempt. Every scenario has a unit binding.
- **Integration and end-to-end levels.** A scenario may carry `@integration-exempt` or `@e2e-exempt`, with a comment
  that gives the reason and the alternative proof.
- **End-to-end on pull requests.** No `test:e2e` runs on a pull request. It runs in the scheduled workflow, so a slow
  suite never blocks a merge.
- **End-to-end for libraries.** Most library feature files carry `@e2e-exempt` tags. The comment states that the library
  is built into the `retriever` binary and has no process, network, or file boundary of its own at that level. The
  alternative proof is `retriever-cli:test:e2e`. No library declares `test:e2e`. Four e2e-tagged test files under
  `libs/*/behaviour/e2e/` are compiled by the `vet` and `build` targets, but no target runs them. Their scenarios are
  not yet run by any gate.
- **Hermetic unit tests.** The architecture guard in `apps/retriever-cli/tests/architecture/hermetic_test.go` reads each
  untagged test file and flags calls to the environment, the disk, processes, sockets, and the real clock.
- **Network denial.** No gate yet runs `test:quick` with the network denied. The gate inventory records this check as
  pending. Until it runs, the guard above is the only enforcement of the unit-level network rule.
