---
description: >-
  Defines a stack pack, the stack standard, local skill, and repository adapter that together govern one language or
  tool in the repository.
when_to_use: >-
  Use when adding a language, framework, or tool to the repository, or when deciding which standard governs a file.
---

# Stack Packs

A stack pack governs one language or tool. It has three parts:

1. a **stack standard** in [development/quality/stacks/](../../development/quality/stacks/README.md), which states the
   choices specific to that stack;
2. a **local skill**, when the stack needs one, placed beside the other skills; and
3. an **extension entry** in `repo-config.yml`, under `extensions`, which declares the projects and gates the stack
   owns.

## Rules

- A standard states only what is specific to its stack. For everything else it links the shared rules: test-driven
  development, the three-level testing standard, and the Nx target standard.
- A stack with no standard is built under the language-neutral rules alone, and the hand-off says so.
- A pack is added in the same change that first adds the stack's code. A pack with no code is not added.
- The [repository adapter](../../development/quality/stacks/repository-adapter.md) states how the repository applies the
  packs it holds.

## Current Packs

| Stack | Standard                                                                 |
| ----- | ------------------------------------------------------------------------ |
| Go    | [Golang Standards](../../development/quality/stacks/golang-standards.md) |
| Shell | [Shell Standards](../../development/quality/stacks/shell-standards.md)   |
