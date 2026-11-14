---
title: Profile layout
description: >-
  PLANNED. Where the files of a work or personal profile live, how a run picks one, and the rules that keep profiles
  apart.
category: reference
tags:
  - reference
  - planned
  - profiles
---

# Profile layout (planned)

**Status: PLANNED.** No profile support exists yet. The reasons are in
[ADR 001](../explanation/decisions/001-user-defined-connectors-and-profiles.md), and the threats are in the
[profile threats](../explanation/decisions/001-user-defined-connectors-and-profiles/004-profile-and-secret-threats.md).

## Folders

A profile is a folder under the user configuration folder. The names below use `$CONFIG` for
`$XDG_CONFIG_HOME/retriever` or `~/.config/retriever`, as [configuration](./retriever-cli-configuration.md) does.

```text
  $CONFIG/profiles/work/sources.yml
  $CONFIG/profiles/work/.env.local
  $CONFIG/profiles/work/connectors/
  $CONFIG/profiles/personal/sources.yml
  $CONFIG/profiles/personal/.env.local
  $CONFIG/profiles/personal/connectors/
```

Each profile has its own catalog, env file, and spec folder. A run reads the folder of the active profile and no other.

## Choosing a profile

1. The `--profile <name>` flag.
2. The `RETRIEVER_PROFILE` environment variable, when it is set and not empty.

A run has exactly one active profile. With no profiles folder, the tool works as it does today, with one catalog. With a
profiles folder and no choice, the tool exits `2` with `retriever.usage.profile_required`. It never picks a default.
`--profile` together with `--catalog` or `--env-file` is a usage error. Tags never select a profile.

## Rules

- A profile name is a lower-case slug of letters, digits, and single hyphens. It is resolved inside `profiles/` only.
- A credential name in a profile catalog starts with the profile name in upper case, with hyphens as underscores. The
  profile `work` uses `WORK_EXAMPLE_API_KEY`. The catalog loader refuses any other name.
- The process environment is read only for names that the active catalog declares.
- A run never lists, reads, or merges a second profile. `mcp` and `serve` keep their profile until they stop.
- `sources list` and the MCP `list_sources` result name the active profile.
