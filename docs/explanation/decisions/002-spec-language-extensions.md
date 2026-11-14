---
title: "ADR 002: Spec language extensions"
description: Decides to extend the connector spec language with nine closed features and to port all three connectors.
category: explanation
tags:
  - explanation
  - decision
  - planned
---

# ADR 002: Spec language extensions

**Status:** accepted on 2026-11-15, by the owner. It replaces the migration order and pull requests 6 to 12 of
[ADR 001](./001-user-defined-connectors-and-profiles.md). Pull requests 1 to 5 of ADR 001 and the Outline golden (#25)
are built. The rest is planned, and the owner can still veto its design details at review.

## Context

ADR 001 planned to port the Outline, Kiwi, and Jira connectors onto the spec engine. A probe of Outline found that the
current language expresses none of its 66 reads. Nine gaps cause that. A spec read must be GET or HEAD (threat S11), but
every Outline call is a POST. Paging names query parameters, but Outline pages in the body. The replies need required
IDs, a nested record with an allow-list, a first item, merged lists, computed tree fields, and a fallback read. A spec
also cannot use the built-in kind `outline` (S10) or the catalog IDs such as `documents.get`.

## Decision

- **D8. Extend the language.** The spec gains nine closed features: E1 POST reads, E2 body paging, E3 required and
  validated fields, E4 nested records with an allow-list, E5 first item or not found, E6 merged lists with constant
  labels, E7 computed tree fields, E8 one fallback read, and E9 bundled kinds with an ID mapping. Each has a page in
  the [companion pages](./002-spec-language-extensions/).
- **D9. Migrate the connectors.** Outline, Kiwi, and Jira are to run on the engine, and each Go connector module leaves
  the dependency graph in the pull request that ports it. If declared lookups cannot express Kiwi, it stays Go code
  until schema version 2, and the owner is told. The [probes](./002-spec-language-extensions/010-kiwi-probe.md) meet
  that rule for Kiwi, and Jira needs a new ADR for one operation.
- **D10. The line.** Every feature is declarative data. There is no expression, no script, and no loop that a spec
  author writes. Each feature is closed and listed, has a fixed limit, and has a named test. A tenth feature needs a
  new ADR.
- **D11. retriever stays pure.** It never calls an LLM, and it never guards an action. The token's own access is the
  guard, so no `--dry-run`, `--yes`, or `--confirm` returns.

## The tension with ADR 001

Decision D1 of ADR 001 says retriever holds no clever logic. The owner put it as "retriever must not have smart logic".
The recommendation, to keep the three Go connectors so that D1 stays whole, was not taken. The owner chose to extend
the language.

The conflict is real. The engine now computes: it merges lists, flattens a tree, and sends a second request. Smart
logic is a matter of degree, and the engine gets more of it. The bound is this: a spec author still
writes only data, and the engine's own logic is the closed list of nine. The rest of D1 holds in full. D1 reads from now
on as "no logic that a spec author writes, and no logic outside the closed list".

## Options considered

- **Keep the Go connectors.** Recommended, and not taken. It keeps D1 whole, but three modules stay, and a user cannot
  copy their work.
- **Port Outline as a plugin.** Rejected. It moves the logic into code, and the plugin host comes later.
- **An expression or template language.** Rejected again, for threat S1.
- **Port only what fits now.** Not viable. None of the 66 Outline reads fits.

## Consequences

- Any user spec may use E1 to E8, so each is untrusted input. The
  [threat model delta](./002-spec-language-extensions/001-threat-model-delta.md) adds threats L1 to L11.
- Threat S11 is decided again: a POST may be a read, under a strict rule
  ([E1](./002-spec-language-extensions/002-post-reads.md)).
- Threat S10 is relaxed for embedded bundled specs only
  ([E9](./002-spec-language-extensions/005-fallback-and-bundled-kinds.md)).
- The nine features are additive to schema version 1
  ([schema impact](./002-spec-language-extensions/006-schema-and-version-impact.md)).
- Nine language pull requests come before the first port.

## Out of scope

An expression language, the plugin host and profiles (ADR 001 stands), channel connectors, and the
`vault` kind.

## Migration order

The Outline golden (done), the nine language pull requests, the Outline port, the Kiwi gate, the Jira port after a
new ADR, the conformance suite, the plugin host, profiles, and the `v0.3.0` pre-release. See the
[sequence](./002-spec-language-extensions/007-pull-requests-language.md) and the
[ports and the rest](./002-spec-language-extensions/008-pull-requests-ports-and-release.md).
