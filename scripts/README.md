---
title: Repository scripts
description: The wrapper scripts, the public-safety gate, and the coverage gate that gates and Nx targets invoke.
when_to_use: Read this before you add a script here, or when you trace which gate invokes one of these scripts.
---

# Repository scripts

Every file in this folder is a command that a gate in [`repo-config.yml`](../repo-config.yml) or an Nx target invokes,
or a part of such a command. Each file exists because its tool cannot take the file list that a gate passes, or because
the check has its own logic. A script that only renames a working command does not belong here.

## Entries

- [`check-remote-dispatch-refusal.sh`](./check-remote-dispatch-refusal.sh) runs the `remote-dispatch-refusal` gate.
  It checks that the remote-dispatch module is in place and that its refusal path is stated.
- [`format-staged`](./format-staged) runs the `format-staged` gate. It formats the staged files it is given:
  Prettier for Markdown, JSON, and YAML, and shfmt for shell scripts. It never reads Git itself.
- [`go-coverage-gate.sh`](./go-coverage-gate.sh) is the coverage gate of one Go module. The `test:unit` and
  `test:coverage` targets of the app and of each library run it from their own module root. It measures the untagged
  tests of that module against a 99 percent floor, and it reads that module's `scripts/coverage-exclusions.tsv`.
- [`go-coverage-gate_test.sh`](./go-coverage-gate_test.sh) is the self-test of the coverage gate. It runs the gate's
  `--check-only` mode against small fixture modules. The app's `test:unit` target runs it first.
- [`lint-shell`](./lint-shell) runs the `shellcheck` gate. It runs `shellcheck` at warning severity over the shell
  scripts among the paths it is given: `*.sh` and `*.bash` files, plus extensionless files whose shebang names a
  shell.
- [`lint-workflows`](./lint-workflows) runs the `actionlint` gate. It lints every GitHub Actions workflow whenever a
  workflow or a local composite action changes, because a change to one workflow can break a caller of another.
- [`public-safety/`](./public-safety/) holds the public-safety gate. It has the tree, commit-message, and range
  screens, the shape set they read, and the tests that prove them. Its own README explains each file.
