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

## Connector variables

No connector variable exists yet. The connector variables arrive with the command-line tool in PR 2, and this page
and `.env.example` will list them then.

## Names read by repository tooling

- `REMOTE_DISPATCH_HOME`: the local binding that governs dispatch to another machine. The remote-dispatch rule reads
  it, and so does the dispatch procedure. No script reads its value yet. See [Dispatch a task to another
  machine](../how-to/dispatch-a-task-to-another-machine.md).

The variable's value is a local path to a binding that lives outside the repository. The repository never records
the value.
