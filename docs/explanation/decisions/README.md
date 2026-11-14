---
title: Decision records
description: Architecture decision records of super-agent, with the form each record follows.
category: explanation
tags:
  - index
  - explanation
  - decision
---

# Decision records

An architecture decision record (ADR) states one decision, the options that lost, and the consequences. Before this
folder existed, the repository kept no decision records, so the first one starts the numbering.

## Directory Map

- [ADR 001: User-defined connectors and profiles](./001-user-defined-connectors-and-profiles.md) records the choice to
  describe connectors as declarative specs, run by one engine, with an opt-in plugin and separate profiles.
- [Companion pages of ADR 001](./001-user-defined-connectors-and-profiles/) hold its threat model, its module layout,
  and its split into pull requests.

## Form of a record

- A record is named `NNN-<slug>.md`. The number never changes and is never reused.
- A record has a status line, then Context, Decision, Options considered, Consequences, Out of scope, and Migration
  order.
- A record stays within 750 words. Longer detail moves into a folder named after the record, with a `README.md`.
- An accepted record is not rewritten. A later record that changes the decision names the record it supersedes, and the
  old record gets a line that links forward.
- Planned work in a record is labelled planned. The as-built description stays in the reference and explanation pages.
