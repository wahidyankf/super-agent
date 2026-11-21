---
title: Repository scripts
description: The wrapper scripts, the public-safety gate, and the coverage gate that gates and Nx targets invoke.
when_to_use: Read this before you add a script here, or when you trace which gate invokes one of these scripts.
---

# Repository scripts

Every file in this folder is a command that a gate in [`repo-config.yml`](../repo-config.yml) or an Nx target invokes,
or a part of such a command. The one exception is the hook health check, which a contributor runs at bootstrap. Each
file exists because its tool cannot take the file list that a gate passes, or because the check has its own logic. A
script that only renames a working command does not belong here.

## Entries

- [`check-commit-message.sh`](./check-commit-message.sh) runs the `commit-message` gate. It passes the commit message
  the typed input supplies to `commitlint`, from the hook's message file or from a pull-request range.
- [`check-commit-range.sh`](./check-commit-range.sh) runs the `commit-history` gate. It lints each commit of a
  `base..head` range on its own with `commitlint`.
- [`check-hooks-installed.sh`](./check-hooks-installed.sh) is the hook health check. It exits non-zero when
  `core.hooksPath` is not the Husky runtime directory or a hook file is missing or not executable. It is a bootstrap
  check, so no gate runs it. CI installs no hooks.
- [`check-hooks-installed_test.sh`](./check-hooks-installed_test.sh) is the self-test of the hook health check. The
  `check-hooks-installed-tests` gate runs it at pull-request. It builds its own fixture repositories.
- [`check-remote-dispatch-refusal.sh`](./check-remote-dispatch-refusal.sh) runs the `remote-dispatch-refusal` gate.
  It checks that the remote-dispatch module is in place and that its refusal path is stated.
- [`format-staged`](./format-staged) runs the `format-staged` gate. It formats the staged files it is given:
  Prettier for Markdown, JSON, and YAML, and shfmt for shell scripts. It never reads Git itself.
- [`go-coverage-gate.sh`](./go-coverage-gate.sh) is the coverage gate of one Go module. The `test:unit` and
  `test:coverage` targets of the app and of each library run it from their own module root. It measures the untagged
  tests of that module against a 99 percent floor, and it reads that module's `scripts/coverage-exclusions.tsv`. Its
  `go test` run has a timeout of `GO_TEST_TIMEOUT`, `20m` by default.
- [`go-coverage-gate_test.sh`](./go-coverage-gate_test.sh) is the self-test of the coverage gate. It runs the gate's
  `--check-only` mode against small fixture modules, and one full run whose test outlives a one-second timeout. The
  app's `test:unit` target runs it first.
- [`go-placeholder-check.sh`](./go-placeholder-check.sh) is the placeholder ban of one Go module (C18). It fails when a
  test file skips, focuses, or marks a step pending, when an assertion is a placeholder such as `assert.True(t, true)`,
  or when a test function has an empty body. It reads tagged test files too. The `lint` target of each Go module runs
  it from the module root, after `golangci-lint`.
- [`lint-shell`](./lint-shell) runs the `shellcheck` gate. It runs `shellcheck` at warning severity over the shell
  scripts among the paths it is given: `*.sh` and `*.bash` files, plus extensionless files whose shebang names a
  shell.
- [`lint-workflows`](./lint-workflows) runs the `actionlint` gate. It lints every GitHub Actions workflow whenever a
  workflow or a local composite action changes, because a change to one workflow can break a caller of another.
- [`public-safety/`](./public-safety/) holds the public-safety gate. It has the tree, commit-message, and range
  screens, the shape set they read, and the tests that prove them. Its own README explains each file.
