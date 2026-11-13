---
description: >-
  Indexes the governed documents and reading order for Canonical Agents.
when_to_use: >-
  Use when navigating Canonical Agents or locating one of its indexed artifacts.
---

# Canonical Agents

One file per agent, each the complete definition the harness adapters route to. The front matter declares the agent's
identity, its `tier`, the `capabilities` it needs, its `constraints`, and the agents it may `dispatches`. Every harness
translates those declarations into its own permission vocabulary.

Generated adapters live under `.claude/agents/`, `.codex/agents/`, `.opencode/agents/`, `.commandcode/agents/`, and
`.agents/agents/`. Never edit them. Edit the canonical file here, then run `./rhino harness adapters generate` and
`./rhino harness adapters validate`.

## Directory Map

### Software Development

- [swe-api-tester](swe-api-tester.md) — judges a running request-based interface against its contract or by exploration,
  without fixing anything.
- [swe-architect](swe-architect.md) — designs boundaries before a build, reviews them after, and holds the architecture
  review lens.
- [swe-debugger](swe-debugger.md) — repairs failing type checks, lint, and tests at the cause.
- [swe-developer](swe-developer.md) — builds behaviour test-first and applies re-validated findings.
- [swe-orchestrator](swe-orchestrator.md) — decomposes a deterministic goal and dispatches the swe family until its checks
  pass.
- [swe-releaser](swe-releaser.md) — cuts versioned releases, publishes built artifacts, and repins pinned tools through the
  documented workflows.
- [swe-reviewer](swe-reviewer.md) — audits code, command-line interface, and scenario bindings against adopted standards.

### Repository Rules

- [repo-rules-maker](repo-rules-maker.md) — creates repository rules and conventions in `repo-governance/`.
- [rules-checker](rules-checker.md) — audits rules for contradictions, inaccurate references, and duplication.
- [rules-fixer](rules-fixer.md) — repairs re-validated rules findings through Rules Propagation.

### Documentation

- [document-writer](document-writer.md) — creates and updates high-quality documentation.
- [docs-checker](docs-checker.md) — audits documentation for factual accuracy, contradictions, and stale references.
- [docs-fixer](docs-fixer.md) — repairs re-validated documentation findings through Docs Propagation.

### Harness Bindings

- [harness-checker](harness-checker.md) — audits harness bindings for parity and upstream drift; changes nothing.
- [harness-fixer](harness-fixer.md) — repairs re-validated harness drift at the canonical source and regenerates adapters.

### Research

- [web-researcher](web-researcher.md) — researches current, verifiable information from the web in an isolated context.
