---
title: Environment variables
description: The environment variable names super-agent reads or plans to read. Names only; no page holds a value.
category: reference
tags:
  - reference
  - environment
  - secrets
---

# Environment variables

This page lists variable names only. Values never belong in a tracked file.

## Where values go

- `.env.example` is tracked. It holds no variable names yet, so every value it could hold is empty.
- `.env.local` holds real values on your machine. Git ignores it. Copy `.env.example` to `.env.local` before you
  fill in values.
- Any other `.env*` file is refused from staging by the `env-validate` gate.

## Command-line tool variables

The `retriever` command-line tool reads these variables. Each one is optional.

- `RETRIEVER_CATALOG`: the path of the sources catalog. It is used when `--catalog` is not given.
- `RETRIEVER_ENV_FILE`: the path of the env file that holds source keys. It is used when `--env-file` is not given.
- `XDG_CONFIG_HOME`: the base folder for the user configuration. The tool reads `retriever/` inside it. When it is unset
  or empty, the tool reads `~/.config/retriever`.

The tool reads no variable named `NO_COLOR`. Its output has no colour, so it has nothing to switch off.

### Source keys

A source's key variable is named by the catalog, in the `credential_env` field, and a Jira source also names an account
e-mail variable in `identity_env`. The names must match `*_API_KEY` or `*_API_TOKEN`, and `*_EMAIL` for an e-mail. The
example catalog uses these names, which hold no value:

- `EXAMPLE_OUTLINE_API_KEY`
- `EXAMPLE_KIWI_API_KEY`
- `EXAMPLE_JIRA_API_TOKEN`
- `EXAMPLE_JIRA_EMAIL`

Your own catalog names its own variables. See [the catalog format](./retriever-cli-catalog-format.md).

### Test seams

Nine variables exist for the repository's own tests. Each source kind has three: `RETRIEVER_<KIND>_TEST_BASE_URL`,
`RETRIEVER_<KIND>_TEST_CA_FILE`, and `RETRIEVER_<KIND>_TEST_TIMEOUT`, where `<KIND>` is `OUTLINE`, `KIWI`, or
`JIRA`. They
point that kind at a local HTTPS test server, and they accept a loopback host only. Do not set them in a real
environment. They are not a supported setting.

## Names read by repository tooling

- `REMOTE_DISPATCH_HOME`: the local binding that governs dispatch to another machine. The remote-dispatch rule reads
  it, and so does the dispatch procedure. No script reads its value yet. See [Dispatch a task to another
  machine](../how-to/dispatch-a-task-to-another-machine.md).

The variable's value is a local path to a binding that lives outside the repository. The repository never records
the value.
