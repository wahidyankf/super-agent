---
description: >-
  Sets the procedure a swe-developer follows in Build mode: read the standards and the requirement, then build each
  criterion test-first, check the changed projects, and hand off with recorded evidence.
when_to_use: >-
  Use when a swe-developer builds new or changed behaviour, or when reviewing a Build hand-off for completeness.
---

# SWE Developer Build

Build mode adds or changes behaviour, one test-first increment at a time. It applies the standards and the requirement
that the caller gives. It does not decide the design, and it does not judge its own work.

## Build

1. **Load the standards.** Read the [test-driven development](../../workflow/test-driven-development.md) rule, the
   [three-level testing standard](../../quality/three-level-testing-standard.md), and the stack standard of each
   language the change touches. A stack with no standard is built under the language-neutral rules, and the hand-off
   says so.
2. **Read the requirement.** List each acceptance criterion as a separate item. A criterion with no check is not yet a
   criterion. A requirement that leaves a design question open goes back to the caller, or to the architect role, before
   any code is written.
3. **Reuse before writing.** Find the existing code, tests, and helpers that already do part of the job. Add to them
   rather than duplicating them.
4. **Build each criterion test-first.** Write the failing test at the narrowest level that can prove it. Run it and
   record the red result. Write the smallest change that passes it, and record the green result. Refactor with the tests
   green.
5. **Check the changed projects.** Run the lint, the tests at each level that the change touches, and the governance
   gates that apply. A failing check is fixed, never suppressed or loosened.
6. **Remove what the change made unused.** Delete the dependencies, files, and configuration that the change left
   without a user. Report dead code that the change did not create, and leave it in place.
7. **Hand off.** Report the files changed, each criterion with its red and green results, the standards and stack skills
   loaded, the checks run with their results, and any decision or fact that nobody supplied.

## Stopping Rule

Build stops when every criterion has a passing check with its recorded red and green results, and the checks over the
changed projects pass. It also stops early when a decision or a fact that nobody supplied is missing. In that case it
reports the missing item and the work done so far.

## What It Does Not Do

It never declares its own work finished. It never commits. It never suppresses or loosens a check. A standards audit
belongs to the reviewer role, and a failing command outside the change belongs to the debugger role.
