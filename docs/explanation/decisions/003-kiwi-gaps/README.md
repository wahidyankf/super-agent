---
title: ADR 003 companion pages
description: The threat model delta, the ten Kiwi gap decisions, and the pull-request sequence of ADR 003.
category: explanation
tags:
  - index
  - explanation
  - decision
  - planned
---

# ADR 003 companion pages

These pages hold the detail of [ADR 003](../003-kiwi-gaps.md). Everything on them is planned.

## Directory Map

- [Threat model delta](./001-threat-model-delta.md) lists what the Kiwi extensions change in the assets, boundaries,
  and entry points, and the new threats L12 to L21.
- [E18 and E10: the JSON-RPC envelope and the error object](./002-envelope-and-errors.md) covers the id echo check
  (K9) and the error object inside HTTP 200 (K1).
- [E11 and E15: prefixed IDs and built URLs](./003-ids-and-urls.md) covers computed output IDs (K2), parsed input IDs
  (K5), and endpoint-built URLs (K6).
- [E12: trees across requests](./004-trees-across-requests.md) covers the fan-out reads and the parent-link assembly
  (K3).
- [E13 and E14: multi-call reads and Markdown](./005-multi-call-reads-and-markdown.md) covers the ordered steps and
  the declarative Markdown layout (K4).
- [E16, E17, and clean-text](./006-guard-pacing-and-text.md) covers the read-before-write guard (K7), per-call pacing
  (K8), and the text converter (K10).
- [Pull requests 25 to 32](./007-pull-requests.md) gives the language pull requests, the Kiwi port, and the
  retirement of `retriever-connector-kiwi`.
