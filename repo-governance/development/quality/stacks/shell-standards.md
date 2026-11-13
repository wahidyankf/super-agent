---
description: >-
  Sets the standard for shell scripts: a strict prologue, portable constructs, clean shfmt and shellcheck runs, and no
  hidden state. Applies to scripts under scripts/ and the hooks under .husky/.
when_to_use: >-
  Use when writing or changing a shell script, a git hook, or a shell snippet inside a gate.
---

# Shell Standards

Shell scripts hold the hook and gate glue. Keep them short. Anything longer than a screen belongs in Go.

## Prologue

- Start with `#!/usr/bin/env bash` and `set -euo pipefail`.
- Quote every variable expansion. Pass arguments with `"$@"`.

## Style and Checks

- Format with `shfmt`. Lint with `shellcheck`. Both run through `scripts/lint-shell` in the gate.
- A suppressed shellcheck rule carries its reason on the same line.
- Name functions and variables in lowercase. Declare function variables with `local`.
- Test with `[[ ]]` in bash. Never parse the output of `ls`.

## Exit Status

A script follows the floor tier of the [exit status
contract](../../../conventions/command-line-interface/001-exit-status-contract.md): `0` for success, `1` for a negative
result, `2` when it cannot complete, and `125` when it refuses before any work starts.

## Portability

- Use syntax that works on bash 3.2, the version that macOS ships, unless the script states a newer requirement.
- Do not depend on GNU-only flags. Check for each required command with `command -v`, and fail with a clear message when
  one is missing.

## Paths and Values

- Derive paths from the script's own location, never from an absolute path.
- Read machine-specific values from environment variables declared in the template. See [No Machine-Specific
  Values](../../../conventions/security/no-machine-specific-values.md).

## Tests

A script that has logic has a test that runs it against a temporary directory and checks its exit status and output.
