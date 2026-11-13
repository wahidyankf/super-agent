---
description: >-
  Sets the functional core and imperative shell practice, and the small, pure, explicit style that the code and the
  agent definitions follow.
when_to_use: >-
  Use when writing a function that decides something, when a function mixes a decision with an effect, or when reviewing
  a change for purity.
---

# Best Practices

## Functional Core, Imperative Shell

Split code into two parts. The core is pure: the same inputs give the same outputs, with no effects. The shell reads
input, calls the core, and performs the effects that the result requires. A function that decides does not also print,
write, or fetch.

## Pure Functions

- Return a value. Do not mutate an argument or a package-level variable.
- Pass the clock, randomness, and configuration in as values.
- A pure function is easy to test at the unit level without test doubles.

## Small and Explicit

- Keep each function to one decision, and name it for that decision.
- Make every dependency a parameter. Do not hide one in a package variable.
- Prefer an explicit value to an implicit default, as [Explicit Over
  Implicit](../../principles/explicit-over-implicit.md) states.

## Immutable Data

- Build a new value rather than changing a shared one.
- Copy a slice or a map before handing it on, when the receiver might change it.

## Agent Roles

- Give each agent one role. A maker creates content, a checker validates it and reports, and a fixer applies only the
  findings it is allowed to apply. Keep the roles apart.
- Apply a fix automatically only when its confidence is high. Report every other fix.

## Related

- [Hexagonal Architecture](hexagonal-architecture.md) places each piece of code in a layer.
