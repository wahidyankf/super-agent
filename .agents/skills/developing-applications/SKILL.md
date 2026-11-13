---
name: developing-applications
description: >-
  Guides language-agnostic application work: placing code in the right layer, deciding where each error is handled or
  propagated, what a log line carries, where input is validated, and building every behaviour test-first.
when_to_use: >-
  Use when implementing or reviewing application or library code in any language, alongside that language's own skill or
  stack standard.
compatibility: Requires read access to the application source, its tests, and its dependency manifests.
---

# Developing Applications

The standards own the rules.
[Hexagonal Architecture](../../../repo-governance/development/pattern/hexagonal-architecture.md) and
[Functional Core, Imperative Shell](../../../repo-governance/development/pattern/best-practices.md) place code;
[Test-Driven Development](../../../repo-governance/development/workflow/test-driven-development.md) and
[Three-Level Testing Standard](../../../repo-governance/development/quality/three-level-testing-standard.md) govern
tests; [Dependency Bump Policy](../../../repo-governance/development/workflow/dependency-bump-policy.md) governs what is
added; and [Implementation Workflow](../../../repo-governance/development/workflow/implementation.md) orders the work. A
language's stack standard adds its own choices, as [Stack Packs](../../../repo-governance/conventions/structure/stack-packs.md)
defines. This skill covers the judgement those rules leave to whoever writes the code.

## Reading Order

This skill is split into companion parts to keep each file short. Read every part in order before
acting; the parts hold the complete skill body.

1. [Place Code Before Writing It, Give Every Error One Fate, A Log Line Is Evidence for Someone Else, Validate Once, Where Trust Ends, Test First, at the Narrowest Layer](001-place-code-before-writing-it.md)
2. [Prove Host Integrations in Their Host, Before Calling It Ready](002-prove-host-integrations-in-their-host.md)
