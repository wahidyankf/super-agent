---
description: >-
  States what super-agent is for: it amplifies one person who works with many agents, data sources, and channels, at
  work and outside work. It is alpha.
when_to_use: >-
  Use when deciding whether a feature, rule, or design serves the purpose of the repository, or when explaining the name
  and scope of super-agent.
---

# Vision: super-agent

super-agent amplifies the person who juggles many resources and channels, at work and outside work, and who uses many
coding agents to help. It gives that person and their harnesses one catalog of sources and one access model.

## The Name

The human is the super agent. super-agent is not a single catch-all agent. It does not decide for the person and it does
not replace their agents. The name describes an arrangement: one person, orchestrating many agents and many sources.

## What It Does

- **Connectors** read and write one kind of source, such as a document system, an issue tracker, or a messaging channel.
  Each connector checks access before it reads or writes.
- **One catalog** lists the sources a person uses. The catalog lives outside this repository, in the person's own
  configuration.
- **Two surfaces** expose the same connectors: a command-line tool for people and scripts, and a protocol server for
  harnesses. Many agents share the same connectors and the same access ceilings.
- **One output shape** lets a harness combine results from many sources: each result names its source, its identifier,
  its address, and its content.

## Planned, Not Built

These are design directions. They are not part of the current release, and the repository must not describe them as
built.

- User-defined connectors, written as declarative specifications, with an optional plug-in escape hatch.
- Separate profiles for work and personal use, each with its own catalog, credentials, and access ceiling.
- A local notes vault as a source kind.
- Orchestration of the agents themselves, which the software-factory design note describes.

## Boundaries

- The public repository holds no work or personal catalog, no real source, and no data. User files live in the person's
  configuration directory.
- Access levels are ceilings. A connector never does more than the level its source allows.
- The project is alpha. Breaking changes are allowed, and each one is listed in the changelog until a stable 1.0.
