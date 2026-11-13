---
description: >-
  Requires coverage to show whether behaviour is tested, not how many lines ran, with a high bar for the core and a
  stated exception for code that no instrument measures reliably.
when_to_use: >-
  Use when setting a coverage target, reading a coverage report, or deciding whether a file may be left out of
  measurement.
---

# Meaningful Coverage

Line coverage counts lines that ran. It does not show that a test checked anything. Coverage here means that each
behaviour in the core is exercised and asserted.

## Targets

- The core packages, which decide behaviour, hold at least 99% statement coverage.
- Other packages hold the target set in their project configuration. No package is left out of measurement to meet a
  target.

## Rules

- Read a coverage number together with the test list. A high number backed by weak assertions is a defect in the tests.
- An exclusion names the file and the reason. An exclusion made for convenience is refused.
- Where no reliable instrument exists, the stack's native verification stands in its place. Examples are generated code
  and a language whose coverage tool misreports. Native verification means validation, lint, the native tests, and
  review.
- Report coverage per level, so a gap at the unit level is not hidden by an end-to-end run.

## Related

- [Test-Driven Development](../../workflow/test-driven-development.md) writes each test before the code it checks.
- [Three-Level Testing Standard](../three-level-testing-standard.md) defines the levels.
