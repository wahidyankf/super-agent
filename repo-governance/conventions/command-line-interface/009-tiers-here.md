---
description: >-
  Records the tier of super-agent-cli under the command-line interface convention, the owner choices that fix its open
  points, and how the shared cross-repository standard will replace this local text.
when_to_use: >-
  Use when building, reviewing, or changing super-agent-cli's interface, or when a convention module seems to conflict
  with what the tool does.
---

# Tiers Here: super-agent-cli

## Tier

`super-agent-cli` is the full bar. A person and a harness call it directly, so it owes the floor rules and the full-bar
rules in modules 001 to 008. It starts no child process, so it never returns `126` or `127`.

## Owner Choices

The owner fixed these points on 2026-11-13. Each one applies to `super-agent-cli` now.

1. **JSON mode is a global flag, `--output text|json`.** It applies to every command. No command has its own JSON flag,
   and no command treats `--json` as an alias.
2. **The failure body on stderr has one shape for every command:** `{"schemaVersion", "command", "exitCode", "error":
{"code", "message", "retryable"}}`. The `error` object adds `field` when one input is at fault. Module 004 gives the
   minimum; this is the superset every command emits.
3. **Runtime exit statuses:** `124` for a bounded wait that elapsed (retryable); `125` for an environment that refuses
   before work starts; `2` for usage, configuration, or internal failure; `1` only for a negative result; `128+N` for a
   signal.
4. **`NO_COLOR` suppresses colour only when it is set and non-empty.** An empty value does not suppress colour.
5. **`--version` prints the tool name and the version.** A panic exits `2` with a coded body, for example
   `super-agent.panic`, and prints no stack trace.

## Conformance Runs Before Merge

The conformance runner for this tool is a quick gate that runs before a pull request merges. It checks the published
contract: exit statuses, error codes, command paths, flags, JSON shapes, and help text. Each change that alters the
contract must update the contract record and the runner in the same pull request.

The runner is not part of this pull request. It arrives with the command-line tool in a later pull request, and until
then the clause is not yet met.

## Shared Standard

The sibling command-line tools in the same group are meant to share one standard. The unified cross-repository standard
will be authored in the shared rules catalog, then adopted here by explicit copy. When that happens, the copy replaces
modules 001 to 008 and this record is updated in the same change. Until then, this repository keeps the local text and
the choices above.
