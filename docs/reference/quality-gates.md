---
title: Quality gates
description: Every quality gate id declared in repo-config.yml, with the surfaces it runs on and what it checks.
category: reference
tags:
  - reference
  - quality
  - gates
---

# Quality gates

Every gate below is declared in the `gates` section of `repo-config.yml`, and RHINO runs it. A surface is the moment
a gate runs: `pre-commit`, `commit-msg`, `pre-push`, or `pull-request`. See [Run the quality
gates](../how-to/run-the-quality-gates.md) for how to run them.

## Public safety

- `public-safety-tree` runs on pre-commit, pre-push, and pull-request. It screens the tracked tree and its file
  names for secrets and private data. It runs `scripts/public-safety/check.sh`.
- `public-safety-commit-message` runs on commit-msg and pull-request. It screens the message being written, and each
  pull-request commit message.
- `public-safety-range` runs on pre-push and pull-request. It screens each commit in a pushed or pull-request range
  on its own additions.

## Harness adapters, dispatch, and environment

- `harness-adapters` runs on pre-push and pull-request. It checks that each generated harness adapter matches the bytes
  that generation would write from its canonical source, so no adapter is edited by hand.

- `remote-dispatch-refusal` runs on pre-commit, pre-push, and pull-request. It checks that the remote-dispatch
  module is in place and that its refusal path is stated. It runs `scripts/check-remote-dispatch-refusal.sh`.
- `env-validate` runs on pre-commit, pre-push, and pull-request. It checks the declared environment policy and that
  no real `.env*` file is staged.

## Configuration and governance

- `repo-config` runs on pre-commit and pull-request. It checks that `repo-config.yml` is valid under the RHINO
  configuration contract.
- `governance-vendor` runs on pre-commit and pull-request. It checks that governance files do not name vendors or
  tools on the forbidden list.
- `convention-emoji` runs on pre-commit and pull-request. It checks that declared code and configuration files hold
  no emoji.

## Markdown

- `format-staged` runs on pre-commit and pull-request. It formats staged Markdown, JSON, YAML, and shell files. It
  changes files, so it is a mutation gate.
- `markdownlint` runs on pre-commit and pull-request. It checks every Markdown file with `markdownlint-cli2` and the
  repository configuration.
- `md-readme-index` runs on pre-commit and pull-request. It checks that the root README of each declared tree links
  each of its direct children.
- `md-internal-link` runs on pre-commit and pull-request. It checks that every relative Markdown link resolves to a
  file or directory in the repository.
- `md-heading-hierarchy` runs on pre-commit and pull-request. It checks that each declared file has one H1, and that
  no heading level is skipped.
- `md-naming` runs on pre-commit and pull-request. It checks that Markdown file names are kebab-case, except the
  names the configuration exempts.
- `md-frontmatter` runs on pre-commit and pull-request. It checks that front matter follows the rules for each
  declared tree.
- `word-budget` runs on pre-commit and pull-request. It checks that governance files and README files stay within
  their word budgets.

## Shell and workflows

- `shellcheck` runs on pre-commit and pull-request. It runs `shellcheck` at warning severity over changed shell
  scripts. It runs `scripts/lint-shell`.
- `actionlint` runs on pre-commit and pull-request. It lints GitHub Actions workflows when a workflow or a local
  action changes. It runs `scripts/lint-workflows`.
