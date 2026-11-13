---
description: >-
  Orders the work on one change: define the goal and its check, make it work, make it right, and make it fast only when
  a measured need exists.
when_to_use: >-
  Use when starting a change, planning the order of its steps, or deciding whether a step is done.
---

# Implementation

Work on one change follows four steps, in order. Each step ends with a check.

1. **Define the goal.** State the outcome and the check that proves it. A goal with no check is not yet a goal.
2. **Make it work.** Write the smallest change that passes the check, test-first, as the [test-driven
   development](test-driven-development.md) rule requires.
3. **Make it right.** Refactor for clarity and for the layering rules, with the tests still green.
4. **Make it fast,** but only when a measured need exists. Measure first, optimise the measured path, and keep the
   check.

## Surgical Changes

- Touch only what the goal needs. Do not reformat, rename, or move unrelated code.
- Remove the code, imports, and files that the change itself made unused.
- Report dead code that the change did not create. Leave it in place.
- Stop when the goal's check passes. Do not add features the goal did not ask for.

## Related

- [Minimal Sufficiency](../../principles/minimal-sufficiency.md) explains why the smallest change is the right one.
- [Root Cause Orientation](../../principles/root-cause-orientation.md) explains how to fix the cause, not the symptom.
