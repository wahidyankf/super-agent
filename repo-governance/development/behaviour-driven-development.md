---
description: >-
  Writes behaviour once as Gherkin and runs it at three levels, unit, integration, and end-to-end, with godog as the
  runner, strict steps, and exemption tags only for integration and end-to-end scenarios with a stated reason.
when_to_use: >-
  Use when writing or changing a feature file, a step definition, or a scenario tag, or when deciding the level a
  scenario runs at.
---

# Behaviour-Driven Development

Behaviour is written once, as Gherkin in `specs/behaviours/`, and the same corpus runs at three levels. The corpus is
the contract for the command line, for the protocol server, and for each connector.

## One Corpus, Three Levels

- **Unit.** The scenario exercises a decision with its dependencies replaced by test doubles. It runs in-process,
  quickly, on every change.
- **Integration.** The scenario exercises an adapter against a real local resource, such as a temporary directory or a
  local stub server. It runs on every pull request.
- **End-to-end.** The scenario exercises the built binary through its public boundary. Keep these few. They run in the
  quality gate before merge.

A scenario carries a tag for each level it must run at. A scenario that runs at every level needs no exemption.

## Runner

- Scenarios run with [godog](https://github.com/cucumber/godog), the Cucumber runner for Go, pinned at v0.16.0 in the
  app's `go.mod`.
- Steps are strict. An undefined or pending step fails the run, and a pending step never counts as a pass.
- A step definition is a thin adapter. The logic it calls lives in the code under test.

## Exemptions

A scenario may carry an exemption tag only at the integration or end-to-end level. The tag's comment states the reason
the scenario cannot run at the skipped level. A unit scenario has no exemption. An exemption is reviewed whenever its
file changes, and it lapses when its reason no longer holds.

## Writing Scenarios

- Write from the outside: the command typed, the input given, and the output and exit status observed.
- Keep one behaviour in each scenario. Use `Scenario Outline` for a table of inputs that share one behaviour.
- Name each scenario so that a failure reads as a sentence.
- Use synthetic data only, as the [public repository data
  policy](../conventions/security/public-repository-data-policy.md) requires.

## Related

- [Test-Driven Development](workflow/test-driven-development.md) sets the cycle that writes each scenario's test first.
- [Three-Level Testing Standard](quality/three-level-testing-standard.md) sets what each level may touch.
- [Specs Directory Structure](../conventions/structure/specs-directory-structure.md) places the feature files.
