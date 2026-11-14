---
description: >-
  Requires super-agent's architecture to be recorded as C4 specifications in specs/architecture, describing what has
  been built, drawn as ASCII diagrams with prose.
when_to_use: >-
  Use when a change alters a container, a component, a boundary, or how one part relates to another, and when reviewing
  an architecture description.
---

# Architecture Specifications

`specs/architecture/` is the as-built model of super-agent. It uses the C4 model: system context, containers, and
components. The model says what exists now.

## Rules

- Describe what has been built. A boundary drawn because it was planned, and never built, misleads the next design
  decision. Label a planned item as planned, or leave it out.
- Draw each diagram as an ASCII block in a `text` fence, as the [diagrams
  convention](../conventions/writing/diagrams.md) requires.
- Put every relationship a diagram shows in the prose beneath it.
- Record each constraint that shapes the design in the model, not only in code. Examples are "the core never reads the
  network" and "the gate holds no defaults about the work it guards".
- Update the model in the same change as the structure it describes. A pull request that adds a container or a component
  updates the model.

## Levels

- **Context.** super-agent, its users (people and harnesses), and the sources it reaches.
- **Containers.** The command-line tool, the protocol server when it exists, the connector engine, and the configuration
  files.
- **Components.** The parts inside one container, such as the command adapter, the source service, and each connector
  kind.

## Review

A reviewer compares each diagram with the code's entry points and imports. A diagram that disagrees with the code is a
defect, in the same way that a failing test is a defect.

The folder layout is in [Specs Directory Structure](../conventions/structure/specs-directory-structure.md).
