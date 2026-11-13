---
description: >-
  Sets the word budgets for governance, instruction, and README files, and the one sanctioned way to fix a file that is
  over budget.
when_to_use: >-
  Use when a governance or instruction file is near or over its word budget, or when a budget is proposed for change.
---

# Word Limit

Coding-agent harnesses load some instruction files before the first user message. Past a harness limit they truncate or
ignore what follows. A word budget keeps each of these files short enough to be read in full.

## Budgets

The budgets live in `repo-config.yml` under `policies.governance.word-budget`. The gate counts whitespace-separated
words in the whole file, with no exclusions.

- Governance, agent, and instruction files, with a target of 650 words and a fail value of 750:
  `repo-governance/**/*.md`, `agents/**/*.md`, `AGENTS.md`, and `CLAUDE.md`.
- READMEs, with a target of 900 words and a fail value of 1000: the root `README.md` and each governed tree's README,
  such as `repo-governance/**/README.md`, `agents/**/README.md`, and `scripts/README.md`.

A file at or below its target has no finding. A file above target and at or below the fail value produces a warning. A
file above the fail value blocks the gate. The budgets are ceilings, not goals. A short file is not worse than a long
one.

## Fixing an Over-Budget File

Apply [minimal sufficiency](../../principles/minimal-sufficiency.md) first: remove what the reader does not need. If the
file still holds reachable detail, use progressive disclosure. Keep the rule in the file, move the detail into a linked
page in a folder named after the file, and give that folder a README.md index.

These fixes are forbidden:

- deleting a rule to meet the number;
- compressing text into dense shorthand that a reader cannot follow;
- splitting the content into another file that harnesses also load automatically; and
- linking to a page that does not exist.

## Changing a Budget

A budget changes only as a class-wide decision, backed by evidence that the limit itself is wrong. Never raise a budget
to let one file pass. Record the reason as a YAML comment beside the value in `repo-config.yml`, then run the gate.

## Enforcement

The `word-budget` gate runs at pre-commit and on the pull-request surface. Run it locally through HIPPO:

```text
./hippo run --class ephemeral --resource-tier light --disk-path . -- ./rhino governance word-budget validate
```
