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
- [ADR 002: Spec language extensions](./002-spec-language-extensions.md) records the choice to extend the spec language
  with nine closed features and to port Outline, Kiwi, and Jira onto it. It replaces the migration order of ADR 001.
- [Companion pages of ADR 002](./002-spec-language-extensions/) hold its threat model delta, its nine extensions, its
  version impact, and its pull-request sequence.
- [ADR 003: Kiwi gaps K1 to K10](./003-kiwi-gaps.md) records the closed extensions that let the Kiwi connector run on
  the spec engine, and the port that retires its Go module. It overrules the Kiwi stop rule of ADR 002.
- [Companion pages of ADR 003](./003-kiwi-gaps/) hold its threat model delta, its ten gap decisions, and its
  pull-request sequence.
- [ADR 004: Jira gaps J0 to J8](./004-jira-gaps.md) records the closed extensions that let the Jira connector run on
  the spec engine, and the port that retires its Go module. It overrules the Jira stop rule of ADR 002.
- [Companion pages of ADR 004](./004-jira-gaps/) hold its threat model delta, its nine gap decisions, and its
  pull-request sequence.

## Form of a record

- A record is named `NNN-<slug>.md`. The number never changes and is never reused.
- A record has a status line, then Context, Decision, Options considered, Consequences, Out of scope, and Migration
  order.
- A record stays within 750 words. Longer detail moves into a folder named after the record, with a `README.md`.
- An accepted record is not rewritten. A later record that changes the decision names the record it supersedes, and the
  old record gets a line that links forward.
- Planned work in a record is labelled planned. The as-built description stays in the reference and explanation pages.
