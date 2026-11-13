---
description: >-
  States that behaviour, dependencies, defaults, and contracts must be stated where they are used, not inferred from
  convention, location, or a hidden default.
when_to_use: >-
  Use when designing a configuration, an interface, a default, or a dependency, or when reviewing code that relies on an
  unstated rule.
---

# Explicit Over Implicit

A reader should find every behaviour that matters in the place they look. Implicit behaviour, such as a default that
nobody wrote down, a rule inferred from a directory name, or a dependency reached through a global, makes the system
harder to trust and harder to change.

## What Must Be Explicit

- **Dependencies.** A function receives what it uses as a parameter. It does not reach for a global.
- **Defaults.** A default is written in the contract, where a reader can see it and change it.
- **Configuration.** Each setting is declared in one place and named for what it does.
- **Contracts.** An exit status, an error code, or a field means what its documentation says, and nothing more.
- **Ownership.** Each file and rule has a named owner or stack, not a guess from its location.

## Why It Matters Here

Many agents and harnesses read this repository. An implicit rule is one that some agents will apply and others will
miss, and the two will drift apart without anyone noticing.

## Trade-Offs

Explicit does not mean verbose. A short, named constant beats a repeated literal. A declared default beats a long list
of values that each repeat it. The test is whether a reader can see the behaviour without guessing.

## Related

- [Fail Closed](fail-closed.md) applies the same idea to missing input: a control refuses rather than guessing.
- [Consumer Independence](consumer-independence.md) keeps consumer-specific values out of shared tools.
