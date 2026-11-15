---
title: "ADR 004: Jira gaps J0 to J8"
description: The closed extensions that move Jira onto the spec engine, and the port that retires its Go module.
category: explanation
tags:
  - explanation
  - decision
  - planned
---

# ADR 004: Jira gaps J0 to J8

**Status:** decided on 2026-11-15, by the owner. It overrules the Jira stop rule of D9 in
[ADR 002](./002-spec-language-extensions.md) for Jira only.

## Context

The [Jira probe](./002-spec-language-extensions/011-jira-probe.md) named nine gaps, J0 to J8, between
`libs/retriever-connector-jira` and features E1 to E9. D9 kept Jira as Go code until a new ADR. The owner
decided: write this ADR for J0 to J8, then port Jira. Each extension below was reassessed against the built
language: E1 to E9 plus page-all, `fallback_pointer`, `unavailable_message`, Icon, URLList, EventList, editor
text, `bool_map`, and `markdown-from-adf`.

## Decision

- **D15. Extend the language again.** Six closed features, E19 to E24, plus three closed converters: one
  row per gap below, one page per feature in the [companion pages](./004-jira-gaps/). The line of D10 holds:
  closed list, fixed limits, named tests.
- **D16. Port Jira.** Jira runs on the engine from a bundled `jira.yml`. D9 is overruled for Jira.
- **D17. Retire the module.** `retriever-connector-jira` leaves the graph in the retirement pull request, with
  an architecture test that fails on any new import. Breaking changes reach the CHANGELOG at port time, not here.

| Gap | What the Go code does                    | Verdict | Feature         |
| --- | ---------------------------------------- | ------- | --------------- |
| J0  | `IssueKey` path parameters               | extend  | E19 `issue_key` |
| J1  | `isLast` decides whether more follows    | extend  | E20 `page_end`  |
| J2  | Over-limit replies cut to the limit      | extend  | E20 `page_end`  |
| J3  | `projectKey` from the issue key          | extend  | E24 `built_ids` |
| J4  | Timestamps rewritten as RFC 3339 in UTC  | extend  | E21 converters  |
| J5  | Titles cut to one line of 255 runes      | extend  | E21 converters  |
| J6  | Markdown assembled from many fields      | extend  | E22 `issue_doc` |
| J7  | The reply key must equal the request key | extend  | E23 `key_echo`  |
| J8  | Document and collection IDs, browse URL  | extend  | E24 `built_ids` |

Nothing is deferred. J1 and J2 share one page-end rule; J3 rides the built-ID check of J8. Already
covered needs no feature: `whoami`, E1, E2, E3, plain nested pointers, per-item `markdown-from-adf`, page-all,
E10 `not_found` names, the error table, and E17 pacing.

## The tension with D1

D1 keeps the engine free of clever logic. This ADR moves in more: page-end decisions, converters, a
document layout, and built IDs. The bound is unchanged: a spec author still writes data, and the engine's logic
stays the closed list E1 to E24 plus the closed converters.

## Options considered

- **Keep the Go module.** Rejected: the owner decided to write this ADR, then port.
- **A Jira-shaped Markdown converter for J6.** Rejected. It would bake one service into the engine, as in
  ADR 003. E22 is generic, with closed line kinds.
- **A free pattern or expression for keys, dates, or titles.** Rejected, for S1. Every keyword stays closed.
- **Failing loud on unparseable dates.** Rejected. The scenarios pin the empty text, so a broken date keeps
  its row with an empty, visible field.

## Consequences

- Any user spec may use E19 to E24 and the three converters, so each is untrusted input. The
  [threat model delta](./004-jira-gaps/001-threat-model-delta.md) adds threats L22 to L28.
- The schema stays at version 1: every feature is a new optional field, and the unknown-field rule still
  protects an older reader. One minor bump for `retriever-spec` and `retriever-engine`.
- Four language pull requests, then the port and the retirement
  ([sequence](./004-jira-gaps/006-pull-requests.md)). The port runs the unchanged Jira scenarios. Failures
  stay loud: `unexpected_response`, or a refusal naming the path.

## Out of scope

The expression language, the plugin host, profiles, channel connectors, and the `vault` kind.

## Migration order

Pull requests 33 to 36 grow the language, 37 ports Jira, and 38 retires the module. The conformance suite,
the plugin host, profiles, and the `v0.3.0` pre-release follow, as ADR 002 orders them.
