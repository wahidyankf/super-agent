---
title: Scripts overview
description: A short summary of each script under scripts/, with links to the full annotations.
category: reference
tags:
  - reference
  - scripts
---

# Scripts overview

Every script under `scripts/` is either a gate command or part of the public-safety gate. The [scripts
index](../../scripts/README.md) annotates each file and directory. This page is the short version.

- `check-remote-dispatch-refusal.sh` runs the `remote-dispatch-refusal` gate. It checks that the remote-dispatch
  module and its refusal path are in place.
- `format-staged` runs the `format-staged` gate. It formats the staged files by extension: Prettier for Markdown,
  JSON, and YAML, and shfmt for shell.
- `lint-shell` runs the `shellcheck` gate over the shell scripts it is given.
- `go-coverage-gate.sh` is the coverage gate of one Go module. The app and each library run it from their own module
  root, and each module is held to its own 99 percent floor. It reads that module's `scripts/coverage-exclusions.tsv`.
- `go-coverage-gate_test.sh` is the self-test of the coverage gate. It runs the gate's `--check-only` mode against
  small fixture modules. The app's `test:unit` target runs it.
- `lint-workflows` runs the `actionlint` gate over every GitHub Actions workflow when one changes.
- `public-safety/` holds the public-safety gate: the tree, commit-message, and range screens, plus their tests.

Read the [public-safety README](../../scripts/public-safety/README.md) before you change anything in
`scripts/public-safety/`. It explains the surfaces, the exit codes, and what the scan never prints.
