---
title: Software-factory note
description: A design note on later orchestration of many agents. Out of scope now; planned items are marked planned.
category: explanation
tags:
  - explanation
  - design
  - planned
---

# Software-factory note

This page is a design note. It records a later direction for super-agent. Nothing on this page is built, and the
direction is out of scope for the current release line. Items marked **planned** are intended, not implemented.

## The idea

super-agent gives you and your agents one catalog of sources and one credential model. The later step is to orchestrate
the agents themselves: to coordinate many agents, so that one agent's output can feed another agent's work. The
project calls that direction a software factory.

Connectors are the data layer under it. Orchestration builds on the same connectors and credentials that the
harnesses already use, so it does not need a second set of credentials.

## Status

- **Out of scope now.** There is no code, no specification, and no plan for the orchestration itself.
- **Not decided.** The exact spec format, the orchestration model, and how it relates to the CLI and the MCP server.

## Planned items

Each item below is planned. None is built.

1. **User-defined connectors (planned).** A user describes a connector in a declarative spec. One engine runs every
   spec, and a conformance suite checks each one. A stdio plugin is an opt-in escape hatch for what a spec cannot
   express.
2. **Work and personal profiles (planned).** Each profile has its own catalog and its own credentials, so its own API
   tokens. One profile is active per run. Tags are labels only, and a tag never switches a profile.
3. **A `vault` kind (planned).** A built-in kind for a local Markdown or Obsidian vault. It uses a root path instead
   of an endpoint and needs no credential. It can list, search, and read notes, resolve wiki links and backlinks,
   and create or update notes inside allowed paths. The vault path lives in the user's catalog, outside the
   repository. The first release writes Markdown to an output path only; the vault kind follows in a later release.
4. **Channel connectors (planned).** A channel connector sends messages as well as reading documents. No channel
   connector ships in the first release. A user can write one as a user-defined connector once item 1 exists.

## Related

- [What super-agent is](./what-super-agent-is.md) describes the scope of the project today.
- [Public data policy](./public-data-policy.md) applies to every planned item, including the vault path.
