---
title: Scripts overview
description: A short summary of each script under scripts/, with links to the full annotations.
category: reference
tags:
  - reference
  - scripts
---

# Scripts overview

Every script under `scripts/` is either a gate command, a hook health check, or part of the public-safety gate. The
[scripts index](../../scripts/README.md) annotates each file and directory. This page is the short version.

- `check-commit-message.sh` runs the `commit-message` gate. At `commit-msg` it reads the message file Git passes, and at
  the pull-request surface it reads each commit of the range.
- `check-commit-range.sh` runs the `commit-history` gate. It lints each commit of a base-to-head range on its own.
- `check-hooks-installed.sh` is the bootstrap check for the Git hooks. It exits `0` when `core.hooksPath` names
  `.husky/_` and the three hook files are present and executable. It is not a pull-request gate.
- `check-hooks-installed_test.sh` is the self-test of that check. It builds its own fixture repositories, and the
  `check-hooks-installed-tests` gate runs it on the pull-request surface.
- `check-remote-dispatch-refusal.sh` runs the `remote-dispatch-refusal` gate. It checks that the remote-dispatch
  module and its refusal path are in place.
- `format-staged` runs the `format-staged` gate. It formats the staged files by extension: Prettier for Markdown,
  JSON, and YAML, and shfmt for shell.
- `lint-shell` runs the `shellcheck` gate over the shell scripts it is given.
- `go-coverage-gate.sh` is the coverage gate of one Go module. The app and each library run it from their own module
  root, and each module is held to its own 99 percent floor. It reads that module's `scripts/coverage-exclusions.tsv`.
  Its `go test` run has a timeout of `GO_TEST_TIMEOUT`, `20m` by default.
- `go-coverage-gate_test.sh` is the self-test of the coverage gate. It runs the gate's `--check-only` mode against
  small fixture modules, plus one full run whose test outlives a one-second timeout. The app's `test:unit` runs it.
- `lint-workflows` runs the `actionlint` gate over every GitHub Actions workflow when one changes.
- `public-safety/` holds the public-safety gate: the tree, commit-message, and range screens, plus their tests.

Read the [public-safety README](../../scripts/public-safety/README.md) before you change anything in
`scripts/public-safety/`. It explains the surfaces, the exit codes, and what the scan never prints.
