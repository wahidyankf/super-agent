---
description: >-
  Limits coverage exclusions to generated files and test-support programs, records each with its reason in one place,
  and admits any other exclusion only as a documented exemption of named boundary code.
when_to_use: >-
  Use when adding or reviewing a coverage exclusion, deciding whether code may leave the measured set, or writing the
  exemption for boundary code a test cannot reach.
---

# Exclusions and Exemptions

An exclusion removes authored code from the denominator, so each one makes the floor in
[Meaningful Coverage](../meaningful-coverage.md#the-floor) easier to clear. Few kinds are allowed, and each is written
down once.

## Two Allowed Exclusions

- **Generated files.** Each carries a generated-code header, and the generator's own tests cover it.
- **Test-support programs.** Programs that only support tests, such as fixture builders and the coverage tool itself,
  are test code, not production code.

No other kind is excluded.

## One Place, One Reason

Each exclusion sits in the gating run's coverage configuration with its reason beside it. No other document restates the
list; the project README links to the configuration. A second list is a second copy that drifts.

## Any Other Exclusion Needs an Exemption

An exemption is an exclusion documented in that same place. It is valid only when all three hold:

- it names a specific file, function, or branch, never a path pattern;
- the named code is wholly a boundary, such as a resource or a process, with no decision in it, or a branch no input can
  take; and
- a named integration or end-to-end target and scenario exercises the boundary code, while an untakeable branch has
  nothing to exercise.

Code mixing a boundary with a decision is split, so the decision stays under the floor. Difficulty, schedule, and
unfinished work are never reasons. Choosing a core package to measure is a wholesale exemption, and none qualifies.

## Enforcement

Gated by the coverage configuration check, which fails on an exclusion that has no reason, on an excluded path that is
neither generated nor test support nor a named exemption, and on a path pattern in an exemption. Unenforced by decision:
whether an exemption's boundary claim is true is a review judgement, made by the reviewer of the change that adds it.
