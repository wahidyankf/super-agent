---
description: >-
  Lists the surfaces that super-agent does not change without a recorded, versioned break: exit statuses, error codes,
  command paths, flags, JSON shapes, configuration keys, and the connector specification schema.
when_to_use: >-
  Use before changing a command, flag, exit status, error code, JSON field, configuration key, or connector
  specification, or when writing the breaking changes for a release.
---

# Public Contract

super-agent is alpha, and it still has users: harnesses and people who depend on its surface. That surface is a contract
even before 1.0. This page lists what the contract covers and what a change to it requires.

## Protected Surfaces

- **Exit statuses.** The meaning of each status in the [exit status
  contract](../conventions/command-line-interface/001-exit-status-contract.md).
- **Error codes.** Each `retriever.<area>.<reason>` code, its meaning, and its `retryable` value.
- **Command paths.** Every command and subcommand name.
- **Flags.** Every flag name and its meaning, including the reserved names `--output`, `--color`, `--no-input`,
  `--help`, and `--version`.
- **JSON shapes.** Every field of every machine-readable document, and its `schemaVersion`.
- **Configuration keys.** Each key's name and meaning, and the precedence order: flag, environment, file, default.
- **Environment variable names** declared in the template.
- **Connector specification schema.** Once it exists, the fields of a declarative connector specification and what each
  one means.

## Not Protected

- Wording of human-readable help and prose diagnostics, provided that help still lists every status and flag.
- Internal packages, function names, and file layout.
- Anything the changelog marks as experimental.

## Compatible and Breaking Changes

Compatible changes add to the surface: a new command, a new optional flag, a new field, or a new error code. A new error
code is not breaking, because each failure mode gets its own code.

Breaking changes remove, rename, or change the meaning of any protected surface. They also include a new exit status
number, because the status set is closed, and a change to a default that alters what a command does.

## Versioning in 0.x

Releases are `v0.x.y`. A breaking change advances the minor version, so `0.4.2` becomes `0.5.0`. A patch release carries
no breaking change.

Each breaking change is listed in `CHANGELOG.md` under its release, with the old form, the new form, and the step that
migrates a caller. Alpha does not excuse a silent break. A break that is missing from the changelog is a defect.

## Process for a Change

1. Name the surface and the change in the pull request.
2. Update the contract record, the conformance runner, and the scenarios in `specs/behaviours/` in the same change.
3. Add the changelog entry, marked breaking when it is one.
4. Get the owner's approval before merging any breaking change.

The conformance runner runs before merge. A change that alters the contract without updating the runner fails the gate.
