---
description: >-
  States what each test boundary excludes, keeps the layers in separate suites, builds gates from named targets, and
  fixes what the fast gate holds, how gating coverage is measured, and which choices an adopter records.
when_to_use: >-
  Use when classifying a test, defining a project's test targets, composing the fast gate, or deciding where coverage is
  measured and where a slow suite runs.
---

# Test Boundaries and Gates

A test result means something only when the test sits where it claims to. A unit test reading a real file is slow,
order-sensitive, and running in a gate that promised neither.

This standard implements [Reproducibility](../../../principles/reproducibility.md),
[Explicit Over Implicit](../../../principles/explicit-over-implicit.md), and
`automation-over-manual.md`. The three layers, classification by the
strongest boundary a test touches, the loopback choice, and any floor above the minimum are owned by
`002-layers-and-adapters.md`.

## What Each Boundary Excludes

| Layer       | Never                                                                                              |
| ----------- | -------------------------------------------------------------------------------------------------- |
| unit        | touches a real disk, database, environment, clock, random, child process, or network; inject each  |
| integration | reaches an external network, drives a browser, or observes the served origin; uses local resources |
| end-to-end  | reaches past the public boundary, uses real user data, or shares a fixture with another case       |

The unit exclusion covers setup and assertions as well as the subject. End-to-end may use any resource, real secrets
included, with synthetic data and per-case fixtures, since only it proves the deployed system.

## Keep the Suites Apart

Unit, integration, and end-to-end tests live in separate trees or targets, so a gate selects one layer without filtering
by name. Support that never executes on its own, such as contracts, bindings, and fixtures, may be shared.

## Gates Compose Named Targets

Each layer and each static check is its own named target. Aggregate gates compose those names instead of repeating their
commands, because a copied command drifts from the original.

| Target                        | Holds                                                                              |
| ----------------------------- | ---------------------------------------------------------------------------------- |
| lint                          | formatting, style, suspicious-code, and dependency findings (`lint-strictness.md`) |
| type check                    | compiler and static-type findings, leaving no build output behind                  |
| unit, integration, end-to-end | one layer's suite each; the unit run also measures coverage                        |
| static behaviour coverage     | scenario and binding resolution, executing nothing, where scenarios exist          |

## The Fast Gate

The fast gate runs type check, lint, unit with coverage, and static behaviour coverage, in that order, so the cheapest
failure to read comes first. It runs in the pull-request gate for affected projects, never in a hook, which would run it
serially and get bypassed, and at the phase/pre-pull-request boundaries owned by
`001-cycle-and-evidence.md`.
A dedicated end-to-end project's fast gate holds only its type check, lint, and static behaviour coverage.

It never runs an integration or end-to-end suite. Integration runs in the pull-request gate as its own job, and
end-to-end in a scheduled or main workflow, where
`004-compliance-and-reporting.md` places them.

## Coverage Is Measured Where It Gates

Gating coverage comes from the one instrumented unit run that executed the tests, because two runs can disagree. The
floor is a 99% minimum fixed by rule C15 of the
[Test and Gate Contract](../checks/test-and-gate-contract/006-coverage-floor.md), never lowered or made optional; what
coverage may measure and the allowed exclusions are owned by [Meaningful Coverage](meaningful-coverage.md). The fast
gate's unit run enforces the floor.

Lowering a floor or widening an exclusion changes what the repository is willing to ship. It is a change to a gate, made
as `software-quality-enforcement.md` requires.

## Adopter Decisions

| Decision    | Option            | Gains                                 | Costs                                    |
| ----------- | ----------------- | ------------------------------------- | ---------------------------------------- |
| task runner | a task-graph tool | selection and ordering come built in  | one more tool to pin and maintain        |
|             | plain scripts     | nothing beyond the language toolchain | affected selection is maintained by hand |

Record each applicable choice. Each project's README names the commands its targets resolve to, and every omitted target
with its reason.

## Enforcement

A failing gate is fixed at its cause, never weakened or bypassed. Each stack pack's hermetic-unit mechanism, an import
and call ban on unit-test files, gates the unit exclusion, since review misses a clock read hidden in setup; review
applies the rest. The adopter enforces the gate contracts in its own task configuration and pull-request gate.
