---
description: >-
  Requires strict resolution of every scenario step, forbids bindings that pass without exercising the subject, and
  limits an exemption to one scenario at one level with a boundary reason and alternative proof.
when_to_use: >-
  Use when writing or reviewing step bindings, when a step is undefined, ambiguous, or unused, or when proposing that a
  scenario skip a level.
---

# Bindings and Exemptions

A binding connects a sentence to a check. When it can pass without the check running, the scenario becomes documentation
that claims to be a test.

## Strict Resolution

Every step resolves to exactly one binding at each level the scenario runs at. The test run is configured so that each
of these fails it:

- an undefined step;
- an ambiguous step that matches more than one binding;
- a duplicate binding for the same expression; and
- an unused binding that no step matches.

Each expanded row of a scenario outline counts as a scenario of its own. Bindings stay thin, translating the step and
calling the subject, with shared setup and helpers kept in support modules.

## A Binding Must Exercise the Subject

A binding never:

- asserts a value it was handed instead of one the subject produced;
- returns early or does nothing;
- remains a stub or pending placeholder; or
- passes without invoking the subject under test.

Each of these is a defect even when the run is green, and review treats it as a failure.

## Exemptions

An exemption excuses one scenario from one level, and only for a boundary reason: that level cannot reach the behaviour
at all. Every exemption carries a written justification stating that reason and its alternative proof: the scenario's
unit binding, which no exemption can remove, or a binding at the other level when that level also covers the behaviour.

Difficulty, runtime, speed, flakiness, cost, and unfinished work such as a TODO are never valid reasons. Each describes
a problem to fix, not a boundary.

An exemption never applies to a `Feature`, a `Rule`, or a `Background`, never to the unit level, and never removes all
proof of a behaviour. The unit binding is mandatory for every scenario precisely because it is the alternative proof.
An exemption is reviewed whenever its file changes, and it lapses when its reason no longer holds.

A scenario exempt at both the integration level and the end-to-end level is a detail, not contract behaviour: only the
unit level proves it. It moves out of the corpus into a plain test beside the code, per
[Behaviour-Driven Development](../behaviour-driven-development.md#scenario-or-plain-test).

## How an Exemption Is Recorded

This repository records an exemption as a tag on the scenario, with a comment directly above the tag that carries the
reason and the alternative proof:

```gherkin
# Exemption(e2e): <why>; alternative-proof: <where>
@e2e-exempt
Scenario: <scenario name>
```

The integration level uses `@integration-exempt` in the same way. The tag form keeps the reason beside the scenario it
excuses, at the cost of test-level vocabulary in the corpus. The binding check in
[Compliance](002-compliance.md) enforces the format and these limits.
