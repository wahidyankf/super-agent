---
description: >-
  States that every control in this repository refuses when it cannot decide, and that a check which could not run is
  never reported as a check that passed.
when_to_use: >-
  Use when designing a gate, a validator, a scanner, or any control that can be absent, unreadable, or unable to reach a
  verdict.
---

# Fail Closed

A control that cannot decide refuses. It does not guess, default, or pass.

This is the most frequently applied rule in this repository, and it is written here once so that the places applying it
can stop re-arguing it.

## What It Requires

Three distinct outcomes, never two: clean, a finding, and could-not-decide. Collapsing the third into the first is the
failure this principle exists to prevent. The two are indistinguishable to everyone downstream, and one of them is a
lie.

A missing input is a refusal. An unreadable one is a refusal. An unknown value where a closed set was expected is a
refusal. Silence is never consent.

## Where It Is Already Load-Bearing

- [Public Outbound Safety](../conventions/security/public-outbound-safety.md): a scan that failed to run never counts as
  a pass.
- [Exit Status Contract](../conventions/command-line-interface/001-exit-status-contract.md): a failure that cannot be
  classified exits `2`, never `0` or `1`.
- [Quality Gate Contract](../development/workflow/quality-gate-contract.md): a gate with a missing input fails, and a
  gate that cannot run fails its surface.
- [Three-Level Testing Standard](../development/quality/three-level-testing-standard.md): a skipped test level is
  reported as skipped, never as passed.
- [Environment Variable Contract](../conventions/security/environment-variable-contract.md): a variable the program
  reads but the template does not declare is a defect.
- [Multi-Harness Binding](../conventions/structure/multi-harness-binding.md): a capability a harness cannot express
  fails the check.

Six applications, one rule. That count is the argument for stating it here.

## Why Not the Convenient Default

Every failure-open default is chosen to be helpful in the common case, and every one of them is wrong in exactly the
case that mattered: the one where something was already broken.

A validator that treats an unreadable configuration as an empty one reports a clean repository. A scanner that treats a
crashed detector as a clean scan publishes the secret it was installed to catch. A generator that emits a default the
repository never chose replaces the harness's own behaviour with a guess frozen at generation time.

In each case the failure is not that the tool was wrong. It is that the tool was wrong and said it was right, which is
the only outcome from which nobody recovers.

## The Cost Is Real and Is Accepted

Failing closed is louder. It stops work for a missing file, an unreadable path, or a term set that nobody updated. That
noise is the price, and it is paid deliberately. An alarm that fires when it cannot see is annoying. An alarm that goes
quiet when it cannot see is worthless.

Where the noise is genuinely wrong, the fix is to narrow the rule in a reviewed change. It is never to add a bypass. See
[Public Outbound Safety](../conventions/security/public-outbound-safety.md), which makes the same point about the same
temptation.
