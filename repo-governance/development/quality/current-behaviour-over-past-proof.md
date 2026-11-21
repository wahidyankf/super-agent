---
description: >-
  Replaces past proof that can no longer be produced, or that only adds complexity, with a recorded observation of
  current behaviour, without waiving any check that can still run today.
when_to_use: >-
  Use when a plan step check, quality gate, review, audit, or delivery checklist asks for a past run, baseline, record,
  or verdict that no longer exists or would have to be rebuilt.
---

# Current Behaviour Over Past Proof

A check sometimes asks for proof from the past that nobody kept: a red run that was never captured, a baseline taken in
a worktree that is gone, a notice that was never recorded, a propagation step that should have run before the edits, or
a reviewed head that was never written down. That history cannot be recovered honestly.

This standard applies [Minimal Sufficiency](../../principles/minimal-sufficiency.md) to evidence, and keeps
[Fail Closed](../../principles/fail-closed.md): the substitute must still prove the criterion.

## The Rule

When a plan step check, quality gate, review, audit, or delivery checklist asks for past proof that can no longer be
produced, or that would only add complexity to rebuild, do not reconstruct it. Observe the current behaviour directly
instead: run the current suites on the current head, read the live state, or read the current files. Record that
observation as the evidence, with the command, the revision, the time, and the result. Add one line that names the past
proof it replaces and why that proof is unavailable.

Reason: a rebuilt record documents the rebuilding, not the event it claims to show, and it costs effort without adding
certainty. The criterion is about how the work behaves now, and that can always be observed.

Owner: the repository owner, by standing instruction on 2026-11-21. The rule is copied from the
[`ose-rules` standard][source] and adapted to this repository's layout.

## Boundaries

- This substitutes evidence; it never waives it. The current observation must establish the criterion on its own.
- A check that can still run today runs today. Only proof tied to a past moment is substituted.
- [Test-Driven Development](../workflow/test-driven-development.md) still binds while the work happens: each red is
  recorded when it runs. This rule covers only a later check that finds a red record missing.
- Do not re-stage a past state, back-date a record, or present a new run as an old one to fill the gap.
- When the behaviour itself can no longer be observed, this rule does not apply. The check stays open, and the owner
  decides, recorded in the plan's Decisions part per [Plans](../../conventions/structure/plans.md).

## Observations

- **Followed:** the evidence holds a current command, revision, time, and result that establish the criterion, plus one
  substitution line that names the missing past proof.
- **Violated:** a past record is rebuilt or back-dated; a check blocks only because unrecoverable history is missing
  while current behaviour establishes the criterion; or a check that can still run is skipped under this rule.

## Enforcement

**Review only, by decision.** Whether a past proof is truly unrecoverable, and whether a current observation establishes
the criterion, are judgements that no gate or test can settle. The reviewer judges them from the evidence and its
substitution line.

[source]: https://github.com/wahidyankf/ose-rules/blob/725fdc1a786cecd306074100e7e4f4471508b23f/repo-governance/development/quality/evidence/current-behaviour-over-past-proof.md
