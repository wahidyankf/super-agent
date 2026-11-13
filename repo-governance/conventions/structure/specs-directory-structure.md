---
description: >-
  Fixes the layout of the specs/ tree: Gherkin behaviour specifications, C4 architecture specifications, and the
  synthetic fixtures they use, each folder with a README index.
when_to_use: >-
  Use when adding or moving a specification, a feature file, an architecture description, or a fixture.
---

# Specs Directory Structure

`specs/` holds what the system must do and how it is built. Code and tests implement the specs. A change to behaviour
starts in `specs/`, not in the code.

## Layout

```text
specs/
  README.md           index of the tree
  behaviours/         Gherkin feature files, one folder per behaviour area
  architecture/       C4 architecture specifications, as ASCII diagrams with prose
    README.md
  fixtures/           synthetic data that tests and feature files read
    README.md
```

A pull request adds only the folders it needs. A folder that does not exist yet is not listed as if it did.

## Rules

- Only an application owns scenarios. A library or a script keeps plain tests and adds no `behaviours/` folder; the
  scenarios of the application that uses it prove what it contributes. The test project of a built binary implements
  the application's corpus and keeps none of its own.
- A feature file covers one behaviour area and holds one `Feature:`.
- A feature describes contract behaviour at the public boundary: a command, a tool exposed over the protocol server, or
  a connector contract. It does not describe internal functions or other developer details.
- A scenario has a stable name. Renaming a scenario changes the contract, so it follows the [public
  contract](../../development/public-contract.md) rules.
- An architecture specification describes what exists. Anything planned is labelled planned.
- A fixture is small and synthetic. It follows the [public repository data
  policy](../security/public-repository-data-policy.md).
- Each folder has a README.md that indexes its files, with one annotated line for each file.
- Specs change in the same change as the behaviour they describe.

The method for behaviour specifications is in [Behaviour-Driven
Development](../../development/behaviour-driven-development.md). The method for architecture specifications is in
[Architecture Specifications](../../development/architecture-specifications.md).
