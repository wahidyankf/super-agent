---
description: >-
  Fails the behaviour-coverage check on a scenario with a missing binding, an unjustified exemption, or an exemption at
  both integration and end-to-end, and keeps integration and end-to-end scenarios out of commit hooks.
when_to_use: >-
  Use when configuring the binding check, deciding where a scenario check runs, or reading why the check failed.
---

# Compliance

Green bindings prove that steps resolve and pass. They do not prove that a scenario's words match what the code does,
which needs review.

## The Behaviour-Coverage Check

Every scenario is bound at the unit, integration, and end-to-end levels unless it carries an exemption. The static
behaviour-coverage check fails on each of these:

- a scenario with no binding at a level it is not exempt from;
- an exemption with no written justification; and
- a scenario exempt at both the integration and end-to-end levels, which is a detail for a plain test.

The check reads the feature files and the binding tags and executes no scenario. It does not judge whether a binding
tests what its sentence says; review does, after a material change to scenarios or their implementation.

Enforcement: gated by the project's binding-check target, which runs in the pull-request pipeline. Integration and
end-to-end scenarios run in the pipeline and before merge, never in a commit hook, because a slow hook gets bypassed.

## Related

- [Bindings and Exemptions](001-bindings-and-exemptions.md) sets the limits the check applies.
- [Three-Level Testing Standard](../quality/three-level-testing-standard.md) places each level in its gate.
