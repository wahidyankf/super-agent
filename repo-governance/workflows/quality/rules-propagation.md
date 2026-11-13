---
description: >-
  Defines how a rule is stated once, placed in the right file, enforced, and carried to every binding that depends on
  it, and how a conflict between two rules is settled.
when_to_use: >-
  Use when adding or changing a rule, when an instruction file and a governance file disagree, or when a harness binding
  must follow a shared change.
---

# Rules Propagation

A rule is stated once, in `repo-governance/`. Every other file that depends on it links to that statement, or carries a
short summary that points back to it. A rule that exists in two places drifts, and the copies disagree.

## Statement

- State the rule in one place, in the file that owns its topic.
- Write the rule as a requirement with a clear subject, and give its reason in one or two sentences.
- Name the owner of the rule when the rule is a decision, so that a later change knows who must agree.

## Placement

- Governance rules go in `repo-governance/`.
- The root `AGENTS.md` and `CLAUDE.md` are instruction files that harnesses load first. They carry only the rules a
  contributor must see before opening any other file, and each one links to the full rule.
- Agent and skill definitions carry the rule their role needs, and link to the full statement.

## Enforcement

Each rule states how it is enforced: a gate, a test, or review. A rule that no gate, test, or review can check is marked
as such, with the reason. Enforcement is named with the rule, not added later.

## Propagation

When a shared rule changes:

1. Change the statement in `repo-governance/` first.
2. Find every summary and every binding that cites it. Search the instruction files, the agent definitions, and the
   skill definitions.
3. Update each summary and each binding in the same change. The harness adapters are regenerated, never hand-edited, as
   the [multi-harness binding](../../conventions/structure/multi-harness-binding.md) convention requires.
4. Run the gates. The governance gates check the links, the word budgets, and the vendor-neutral wording.

## Conflicts

Two rules that disagree are a defect. The owner decides which one holds. The decision is recorded in the rule that wins,
and the other rule is corrected in the same change. No agent picks a winner on its own, and a conflict is never resolved
by silently following the rule that is more convenient.

## Shared Rules From Outside

A rule that comes from another repository is copied by explicit change, and the copy records its source. Nothing updates
it automatically, so a later reader can compare the copy with the source.

## Related

- [Governance README Completeness](../../conventions/structure/governance-readme-completeness.md) keeps the index
  current when a rule file is added.
- [Quality Gate Contract](../../development/workflow/quality-gate-contract.md) explains how a gate is declared.
