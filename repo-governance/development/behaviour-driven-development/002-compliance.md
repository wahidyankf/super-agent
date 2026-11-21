---
description: >-
  Fails the behaviour-coverage check on a scenario with a missing binding, an unjustified or stale exemption, or an
  exemption at both integration and end-to-end, and keeps the integration and end-to-end runs off the commit hooks.
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
- an exemption with no valid reason and target, and a stale exemption whose level is already bound;
- a scenario exempt at both the integration and end-to-end levels, which is a detail for a plain test.

The check is RHINO's `behaviour coverage validate`. It runs the rules B01 to B09 of the
[behaviour corpus module](../quality/checks/test-and-gate-contract/005-behaviour-corpus.md), and it reads the feature
files and the binding comments. It executes no scenario. It does not judge whether a binding tests what its sentence
says; review does, after a material change to scenarios or their implementation.

Enforcement: the `test:behaviour:coverage` target of each project runs the check, and `test:quick` runs it locally. The
pull-request workflow runs it once for the repository. A failure on any of these fails the pull request.

Integration tests run on the pull request. End-to-end tests do not: they run in the scheduled or main workflow, so a
slow suite never blocks a merge, and no commit hook runs either level.

## Related

- [Bindings and Exemptions](001-bindings-and-exemptions.md) sets the limits the check applies.
- [Test and Gate Contract, module 004](../quality/checks/test-and-gate-contract/004-test-levels-and-targets.md) places
  each level in its gate.
- [Test and Gate Contract, module 005](../quality/checks/test-and-gate-contract/005-behaviour-corpus.md) states the
  binding convention and the rules B01 to B09.
