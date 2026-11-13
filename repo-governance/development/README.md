---
description: >-
  Indexes the development rules: the public contract, behaviour and architecture specifications, the agent delegation
  rules, design patterns, practice rules, quality standards, and the development workflow.
when_to_use: >-
  Use to find the rule for building, testing, specifying, or delivering a change to super-agent.
---

# Development

Development rules say how a change is specified, built, tested, and kept compatible with the people and tools that use
it.

## Directory Map

- [Public Contract](./public-contract.md): the surfaces that a change must not break without a recorded, versioned
  break.
- [Architecture Specifications](./architecture-specifications.md): the C4 model of what has been built, kept in
  `specs/`.
- [Behaviour-Driven Development](./behaviour-driven-development.md): behaviour written once as Gherkin and run at three
  levels.
- [Agents](./agents/README.md): when coding work is delegated to a swe agent, and the procedures those agents follow.
- [Pattern](./pattern/README.md): hexagonal architecture and the functional core, imperative shell practice.
- [Practice](./practice/README.md): resource-aware development for local compute, and task tracking.
- [Quality](./quality/README.md): the three-level testing standard, coverage, Nx target names, and the stack standards.
- [Workflow](./workflow/README.md): test-driven development, implementation order, dependencies, git identity, and
  gates.
