---
description: >-
  Chooses a Gherkin scenario for contract behaviour and a plain test for a developer detail, gives only an application
  a corpus, runs that corpus at three levels with godog, and allows an exemption only at integration or end-to-end.
when_to_use: >-
  Use when writing or changing a feature file, a step definition, or a scenario tag, when choosing between a scenario
  and a plain test, or when deciding the level a scenario runs at.
---

# Behaviour-Driven Development

Behaviour is written once as Gherkin in `specs/behaviours/`. The same examples drive every test level that can prove
them and are the contract for the command line and the protocol servers.

## Scenario or Plain Test

Who notices a break decides how a behaviour is proved:

- **Contract behaviour** changes what a user, operator, or harness sees when it breaks: command-line output, exit codes,
  machine-readable bodies, protocol results, and access and safety rules. It must be a Gherkin scenario in `specs/`.
- **A developer detail** is noticed only by a developer: a branch, an internal helper, a configuration mapping, a
  structure rule. It must be a plain test beside the code, never a scenario.

A detail written as a scenario buries the contract in noise; a contract left as a plain test can change unseen.

A scenario that needs both an integration exemption and an end-to-end exemption is a detail: only the unit level proves
it, so it becomes a plain test, per
[Bindings and Exemptions](behaviour-driven-development/001-bindings-and-exemptions.md).

Enforcement: gated for the exemption pair, by the binding check in
[Compliance](behaviour-driven-development/002-compliance.md). Unenforced by decision for the choice itself, because
whether anyone outside the code sees a break is a review judgement.

## Only Applications Own a Corpus

An application is anything with an entry point a user, operator, or harness reaches, such as a command-line tool or a
protocol server. Each keeps one canonical Gherkin corpus in one directory tree, discovered recursively. A shared library
and a script keep plain tests and no scenarios; the scenarios of the application using them prove what they contribute.
Nothing registers feature files in a list, which would drift.

A test project that runs a built binary implements the corpus of the application it tests. It keeps no scenarios of its
own, so a behaviour never has two competing statements.

## One Corpus, Three Levels

- **Unit.** The scenario exercises a decision with its dependencies replaced by test doubles. It runs in-process, on
  every change.
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

## Writing Scenarios

- Write from the outside: the command typed, the input given, and the output and exit status observed.
- Keep one behaviour in each scenario, and use `Scenario Outline` for a table of inputs that share it.
- Name each scenario so that a failure reads as a sentence.
- Use synthetic data only, as the [public repository data
  policy](../conventions/security/public-repository-data-policy.md) requires.

## Related

- [Bindings and Exemptions](behaviour-driven-development/001-bindings-and-exemptions.md) and
  [Compliance](behaviour-driven-development/002-compliance.md) hold the binding and exemption limits and the check.
- [Test-Driven Development](workflow/test-driven-development.md) sets the cycle that writes each scenario's test first.
- [Three-Level Testing Standard](quality/three-level-testing-standard.md) sets what each level may touch.
- [Specs Directory Structure](../conventions/structure/specs-directory-structure.md) places the feature files.
