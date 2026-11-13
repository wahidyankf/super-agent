---
description: >-
  Indexes the canonical skill source and the agent path policy for this repository.
when_to_use: >-
  Use when navigating the canonical skills, the generated muse agent output, or the agent path policy.
---

# Agent Configuration

This tree holds the canonical skill definitions and the agent path policy. Canonical agent definitions live in `agents/`
at the repository root. Each coding harness carries a generated adapter that declares its identity and imports the
canonical path; the adapter never restates the instruction.

Edit canonical skills under `skills/` and canonical agents under `agents/`. Then run `./rhino harness adapters generate`
and `./rhino harness adapters validate`. Never edit the generated output under `agents/` in this tree.

## Directory Map

- [Skills](skills/README.md) — the canonical skill source, one directory per skill.
- `agents/` — generated muse adapter output (`agents/<id>/AGENT.md`) and its `catalog.json` and `provenance.json`. Never
  edited by hand.
- [agent-policy.json](agent-policy.json) — the protected and permitted path patterns for agent tools. No hook in this
  repository reads it yet.
