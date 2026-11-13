---
description: >-
  Keeps one canonical definition for each agent and skill and generates each harness's adapter from it, so that no
  harness copy is edited by hand.
when_to_use: >-
  Use when adding or changing an agent, a skill, or a harness binding, or when two harnesses disagree about the same
  agent.
---

# Multi-Harness Binding

A harness is a coding-agent runtime that reads its own configuration format. super-agent works with several harnesses,
so one agent definition must serve all of them without drift.

## Canonical Sources

- Each agent is defined once, in `agents/<name>.md`. Its front matter holds the name, the description, the tier, the
  capabilities, and the constraints.
- Each skill is defined once, in `.agents/skills/<name>/SKILL.md`.
- Hand-author only these canonical files. Every harness adapter is generated from them.

## Portable Vocabulary

Capabilities and constraints use a shared vocabulary:

- capabilities: `repository-read`, `repository-write`, `shell`, `network`, and `subagent`;
- constraints: `read-only`, `no-edit`, `no-glob`, `no-read`, and `no-write`.

Each harness profile in `repo-config.yml` maps these names to its own tool names. A capability that a harness cannot
express fails the check. It is never dropped silently.

## Tiers

Each agent names one tier: `ultra`, `plan`, `execution`, or `fast`. `repo-config.yml` maps each tier to a model and an
effort for each harness. A harness with no mapping for a tier emits nothing for that tier and says so. It never guesses
a model, because a guess frozen into an adapter outlives the reason for it.

## Adapters

Each adapter carries a short route line. The route tells the harness to read the canonical file at its repository path,
and to stop and report if the file cannot be read. The route text is the same for every harness.

Adapters are generated and never edited by hand. A change to an agent touches the canonical file, and the generator
updates the adapters in the same change.

## Parity

A parity check compares every adapter with its canonical file and fails on drift. The check runs in the quality gates
before merge. When a harness or a shared rule changes, the rules propagation workflow applies the change to every
binding it affects.

## Adding a Harness

1. Add a profile to `repo-config.yml`, with its adapter paths, its tool mappings, and its tier mappings.
2. Generate the adapters and run the parity check.
3. Record in the hand-off any capability that the harness cannot express.

The rules for changing shared rules across bindings are in [Rules
Propagation](../../workflows/quality/rules-propagation.md).
