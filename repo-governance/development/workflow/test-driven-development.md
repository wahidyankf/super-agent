---
description: >-
  Requires every behaviour change to start from a test that is seen failing, to pass it with the smallest change, and to
  refactor with the tests still green.
when_to_use: >-
  Use when implementing or changing behaviour, when fixing a defect, or when reviewing a change with no test that failed
  first.
---

# Test-Driven Development

Every behaviour change follows the Red, Green, Refactor cycle. The test comes first, and someone must see it fail before
it passes.

## The Cycle

1. **Red.** Write the test for one behaviour and run it. Record what it reported. A test that was never seen failing
   proves nothing.
2. **Green.** Write the smallest change that makes the test pass. Do not add behaviour that the test does not ask for.
3. **Refactor.** Improve the structure while every test still passes. Run the tests after each step.

## Rules

- Start at the narrowest level that can prove the behaviour: unit for a decision, integration for an adapter, and
  end-to-end for a journey. The [three-level testing standard](../quality/three-level-testing-standard.md) sets the
  levels.
- A defect fix starts with a test that reproduces the defect and fails on the current code.
- Do not weaken, skip, or delete a failing test to reach green. A test that is wrong is corrected, and the change says
  why.
- A test checks behaviour through the public interface, not private state.
- Each error path has its own test.

## Batches

A plan may group several cycles into one pull request. Each cycle still records its own red and green results.

## Evidence

The pull request names each new test and the result it recorded when it first ran red.

## Related

- [Behaviour-Driven Development](../behaviour-driven-development.md) gives the Gherkin form of the tests.
- [Implementation](implementation.md) gives the order of the work around each cycle.
