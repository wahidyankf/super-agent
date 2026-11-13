---
description: >-
  Limits numeric coverage to authored executable production code a reliable instrument measures, requires a 99% floor on
  the unit figure over the whole project, and gives integration, end-to-end, and infrastructure code no line target.
when_to_use: >-
  Use when configuring coverage measurement, setting or checking the coverage floor, deciding what a coverage report may
  include, or deciding whether a layer or stack carries a numeric coverage target at all.
---

# Meaningful Coverage

A coverage number is evidence only about the code it measured, and only when the instrument measured it correctly.
Counted over generated files or configuration, or by a tool that misreports, it moves while the untested behaviour stays
exactly where it was. A high number backed by weak assertions is a defect in the tests, so read a number together with
the test list.

This standard decides what coverage may measure and the floor it must reach. Which code may be excluded belongs to
[Exclusions and Exemptions](meaningful-coverage/001-exclusions-and-exemptions.md).

## What a Number May Measure

Numeric coverage applies only where both hold:

- a reliable instrument measures the code, attributing executed lines and branches correctly for that language and
  runtime; and
- the code is authored, executable, production code.

When either fails, a number is not evidence, and no number is reported.

## The Floor

Wherever a number may be measured, a project's unit-test coverage must be at least 99%:

- **Unit tests only.** The figure counts what unit tests execute, the unit-level scenario bindings included. Integration
  and end-to-end tests prove real boundaries, carry no line target, and stay outside the number.
- **Whole project.** The denominator is every authored production file of the project, never a chosen core package. A
  floor over a core can read 99% while the adapters go unmeasured.
- **One run.** Every unit test and unit scenario binding runs in one instrumented run whose profiles form a union, so no
  line counts twice and the figure the gate judges is the figure that run produced.
- **A minimum.** A repository may record a higher floor, never a lower one.

The fast gate's unit run enforces the floor. Lowering the floor or widening an exclusion is a change to a gate, and a
gate is never weakened to make a change pass, per the [quality gate contract](../../workflow/quality-gate-contract.md).

## Outside Numeric Coverage

| Code                       | Why it is outside                                                        |
| -------------------------- | ------------------------------------------------------------------------ |
| generated output           | the generator's own tests cover it; its lines inflate or dilute a figure |
| test-support programs      | they are test code, not production code                                  |
| declarative configuration  | nothing executes line by line; native validation checks it               |
| external and vendored code | its owner tests it, and this repository cannot change it                 |

Only the first two rows are exclusions from an otherwise measured project; the last two never enter the denominator.
Every exclusion and exemption follows [Exclusions and Exemptions](meaningful-coverage/001-exclusions-and-exemptions.md).

## No Surrogate Metric

Where no reliable instrument exists, such as a shell dialect whose coverage tool misreports, the stack's native
verification stands in its place: validation, lint, the native tests, and review. Never invent a surrogate number, such
as scripts with a test, to fill a coverage slot, and never define a coverage target that measures nothing.

## Enforcement

Gated by the coverage gate, which fails below the floor on the unit run's figure and fails when an authored production
file is absent from the measured set. Show it failing on a 98.9% figure or a missing package, and passing at 99.0% with
every package present. Unenforced by decision: whether an instrument is reliable and whether code is authored production
code are review judgements that no validator makes well.

## Related

- [Test-Driven Development](../../workflow/test-driven-development.md) writes each test before the code it checks.
- [Three-Level Testing Standard](../three-level-testing-standard.md) defines the levels.
