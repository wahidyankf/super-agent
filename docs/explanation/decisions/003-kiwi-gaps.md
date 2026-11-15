---
title: "ADR 003: Kiwi gaps K1 to K10"
description: The closed extensions that move Kiwi onto the spec engine, and the port that retires its Go module.
category: explanation
tags:
  - explanation
  - decision
  - planned
---

# ADR 003: Kiwi gaps K1 to K10

**Status:** decided on 2026-11-15, by the owner. It overrules the Kiwi stop rule of D9 in
[ADR 002](./002-spec-language-extensions.md) for Kiwi only; Jira still waits for its own ADR.

## Context

The [Kiwi probe](./002-spec-language-extensions/010-kiwi-probe.md) named ten gaps, K1 to K10, between
`libs/retriever-connector-kiwi` and features E1 to E9. D9 kept Kiwi as Go code until schema version 2. The owner
decided otherwise: write this ADR for K1 to K10, then port Kiwi.

## Decision

- **D12. Extend the language again.** Nine closed features, E10 to E18, plus one closed converter, `clean-text`. Each
  gap gets one row below, and each feature gets a page in the [companion pages](./003-kiwi-gaps/). The line of D10
  holds: the list is closed, each item has fixed limits and named tests, and a further feature needs a new ADR.
- **D13. Migrate all.** Kiwi runs on the engine from a bundled `kiwi.yml`. D9 is overruled for Kiwi; Jira still waits
  for its own ADR.
- **D14. Retire the module.** `retriever-connector-kiwi` leaves the graph in the retirement pull request, with an
  architecture test that fails on any new import. Breaking changes reach the CHANGELOG at port time, not here.

| Gap | What the Go code does                       | Verdict | Feature                        |
| --- | ------------------------------------------- | ------- | ------------------------------ |
| K1  | Error object inside HTTP 200                | extend  | E10 `error_object`             |
| K2  | Output IDs like `product-<n>`               | extend  | E11 `prefix`                   |
| K3  | Tree from parent links across requests      | extend  | E12 `fanout`                   |
| K4  | Multi-call reads with Markdown assembly     | extend  | E13 `assemble`, E14 `markdown` |
| K5  | Input IDs like `plan-<n>` parsed            | extend  | E11 `prefix`                   |
| K6  | URLs built from the endpoint                | extend  | E15 `doc_url`                  |
| K7  | Read-before-write guard                     | extend  | E16 `guard`                    |
| K8  | Per-call pacing                             | extend  | E17 `pacing`                   |
| K9  | JSON-RPC id echo check                      | extend  | E18 `envelope`                 |
| K10 | HTML unescape plus invisible-char stripping | extend  | `clean-text` converter         |

Nothing is deferred. K2 and K5 share one feature, both directions of one canonical rule. The writes' name-to-ID
lookups keep riding `resolve`, so no gap needs a new lookup feature.

## The tension with D1

D1 keeps the engine free of clever logic. This ADR moves in more of it: a fan-out of up to 200 calls, a four-step
assembly, and Markdown rendering. The bound is unchanged: a spec author still writes data, and the engine's logic
stays the closed list E1 to E18 plus the closed converters. Anything outside it is refused by the extended lock.

## Options considered

- **Keep the Go module.** Rejected. The owner decided "migrate all".
- **A code-only error mapping for K1.** Rejected. The scenarios pin message-mapped sentinels such as key rejected on
  "login required", so dropping the word table would change behaviour.
- **Kiwi-shaped Markdown converters for K4.** Rejected. They would bake one service into the engine.
- **A free pattern or expression for IDs, URLs, or errors.** Rejected again, for S1. Every new keyword is closed.
- **Caps enforced by the stub or the review text.** Rejected. Caps that matter are engine-enforced and test-held.

## Consequences

- Any user spec may use E10 to E18 and `clean-text`, so each is untrusted input. The
  [threat model delta](./003-kiwi-gaps/001-threat-model-delta.md) adds threats L12 to L21.
- The schema stays at version 1: every feature is a new optional field, and the unknown-field rule still protects an
  older reader. One minor bump for `retriever-spec` and `retriever-engine`.
- Six pull requests grow the language, one ports Kiwi, and one retires the module
  ([sequence](./003-kiwi-gaps/007-pull-requests.md)). Each port pull request runs the unchanged Kiwi scenarios, so the
  scenarios prove that nothing moved. Failures stay loud: `unexpected_response`, or a refusal naming the path.

## Out of scope

Jira gaps J0 to J8, which need their own ADR. The expression language, the plugin host, profiles, channel connectors,
and the `vault` kind.

## Migration order

Pull requests 25 to 30 grow the language, 31 ports Kiwi, and 32 retires the module. The Jira ADR, the conformance
suite, the plugin host, profiles, and the `v0.3.0` pre-release follow, as ADR 002 orders them.
