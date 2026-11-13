---
description: >-
  Indexes the principles that shape every rule in this repository: minimal sufficiency, fail closed, explicit over
  implicit, root cause, reproducibility, and consumer independence.
when_to_use: >-
  Use to find the principle behind a rule, or to check a design against the repository's basic commitments.
---

# Principles

Principles are the reasons behind the rules. A rule that conflicts with a principle is the rule that is wrong.

## Directory Map

- [Minimal Sufficiency](./minimal-sufficiency.md): every addition must earn its place, and the smallest sufficient form
  wins.
- [Fail Closed](./fail-closed.md): a control that cannot decide refuses, and never reports a pass it did not earn.
- [Consumer Independence](./consumer-independence.md): a shared tool holds none of its consumers' answers.
- [Explicit Over Implicit](./explicit-over-implicit.md): behaviour, defaults, and contracts are stated where they apply.
- [Root Cause Orientation](./root-cause-orientation.md): a defect is fixed at its cause, and a workaround is labelled.
- [Reproducibility](./reproducibility.md): the same inputs give the same result on any clean checkout.
