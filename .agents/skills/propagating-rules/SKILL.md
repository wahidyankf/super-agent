---
name: propagating-rules
description: >-
  Guides recognising rule work, whether named, stated in ordinary words, or implied by the file being edited, and
  routing it through Rules Propagation before the first edit, without softening a rule to make it fit.
when_to_use: >-
  Use the moment a request implies a lasting obligation, or before editing any location that holds rules.
compatibility: >-
  Requires read access to the repository's rule-bearing locations.
---

# Propagating Rules

[Rules Propagation](../../../repo-governance/workflows/quality/rules-propagation.md) owns the sequence, and
[Rule Definition](../../../repo-governance/workflows/quality/rules-propagation.md) owns what a rule is and where rules
live. This skill covers the judgement that comes first: noticing that a request is rule work at all.

## Recognising Rule Work

The wording varies far more than the intent. Treat each of these as rule work:

- **Named.** The request says rule, convention, policy, or standard, and asks to add, change, move, or remove one.
- **Stated in ordinary words.** The request sets a standing expectation without naming it: from now on, always, never
  again, before every release. An obligation stated in prose is a rule whether or not the word appears.
- **Implied by the target.** The edit lands in a rule-bearing location: governance prose, a root instruction file, an
  agent or skill definition, a gate declaration, or the hooks and pipeline jobs that enforce one.

When the target is unclear, ask the two membership questions in Rule Definition, in order. An instruction for the task
in hand fails the second: it does not outlive the work that prompted it, so it is not rule work.

## Start Before the First Edit

Enter propagation before touching any rule-bearing file, not after. Its conflict scan runs before writing because a
contradiction found afterwards usually means the wrong rule was edited, and unpicking that costs more than the scan.

Work already under way that turns out to be rule work stops there and enters propagation with what it has.

## What Not to Do

- Do not soften a rule that cannot be made falsifiable into guidance so that it can be written; that rule halts.
- Do not raise a word budget to fit a rule; a full surface relocates its weakest entry instead.
- Do not leave a second statement of the same obligation standing. Every other place stating the subject is kept,
  amended, merged, removed, relocated, or superseded, and a copy kept on purpose records why.
- Do not read a passing validator as proof that a rule is well placed; placement is a judgement the workflow records.

## Related

- [Governance Layers](../../../repo-governance/README.md) — which level a rule belongs to.
