---
description: >-
  Indexes the structure conventions: the multi-harness binding of agents and skills, the specs tree, related
  repositories, the plans folder, stack packs, and the README completeness rule.
when_to_use: >-
  Use to find the rule for where a file belongs, how an agent reaches each harness, or how the repository relates to its
  group.
---

# Structure Conventions

These conventions fix where things live and how the parts of the repository connect.

## Directory Map

- [Multi-Harness Binding](./multi-harness-binding.md): one canonical definition for each agent and skill, and generated
  adapters for each harness.
- [Specs Directory Structure](./specs-directory-structure.md): the layout of `specs/` for behaviours, architecture, and
  fixtures.
- [Related Repositories](./related-repositories.md): the group membership, what is shared, and what is never shared.
- [Plans](./plans.md): when a plan may be written, its lifecycle folders, and its required content.
- [Stack Packs](./stack-packs.md): the standard, skill, and extension entry that govern one language or tool.
- [Governance README Completeness](./governance-readme-completeness.md): the rule that every governed README links each
  direct child.
