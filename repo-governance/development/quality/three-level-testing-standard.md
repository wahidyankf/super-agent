---
description: >-
  Sets the three test levels, unit, integration, and end-to-end, what each may touch, how fast each must run, and the
  gate each one runs at.
when_to_use: >-
  Use when deciding which level a test belongs to, when adding a test dependency, or when a gate runs the wrong set of
  tests.
---

# Three-Level Testing Standard

Every behaviour is tested at one of three levels. The level sets what the test may touch, how fast it must run, and
where it runs.

- **Unit.** Pure logic and test doubles only. No network, real disk, or clock. It finishes in seconds and runs at
  pre-commit and on every pull request.
- **Integration.** An adapter and a real local resource, such as a temporary directory or a local stub server. It
  finishes under a minute and runs on every pull request.
- **End-to-end.** The built binary, through its public boundary. It finishes in a few minutes and runs before merge, in
  the quality gate.

## Rules

- A unit test receives its clock, randomness, and I/O as parameters. It never reads the real environment.
- An integration test creates the resource it uses and removes it afterwards. It never calls a live external service.
- An end-to-end test covers a journey, not a single unit. Keep the set small. Move a case down a level when it can prove
  the same behaviour there.
- A test that needs a live credential does not belong in this repository. It runs under the owner's control, outside
  pull requests.
- The quality gate reports a skipped level as skipped, never as passed.
- Keep the levels in separate trees or targets, so a gate selects one level without filtering by name. A stack standard
  may select a level by a build tag, and the unit level then carries no tag.
- Gating coverage comes from the one instrumented unit run that executed the tests, because two runs can disagree. The
  floor and the allowed exclusions belong to [Meaningful Coverage](testing/meaningful-coverage.md), and the fast gate's
  unit run enforces the floor.

## Related

- [Behaviour-Driven Development](../behaviour-driven-development.md) gives the corpus that runs at each level.
- [Test-Driven Development](../workflow/test-driven-development.md) gives the cycle that writes each test first.
- [Meaningful Coverage](testing/meaningful-coverage.md) defines the coverage floor and what the numbers measure.
