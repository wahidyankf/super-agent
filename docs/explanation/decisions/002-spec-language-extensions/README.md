---
title: ADR 002 companion pages
description: The threat model delta, the nine extensions, the version impact, and the pull-request sequence of ADR 002.
category: explanation
tags:
  - index
  - explanation
  - decision
  - planned
---

# ADR 002 companion pages

These pages hold the detail of [ADR 002](../002-spec-language-extensions.md). Everything on them is planned.

## Directory Map

- [Threat model delta](./001-threat-model-delta.md) lists what the extensions change in the assets, boundaries, and
  entry points, and the new threats L1 to L11.
- [E1: POST reads](./002-post-reads.md) gives the strict rule that decides threat S11 again.
- [E2 and E3: body paging and validated fields](./003-paging-and-validated-fields.md) covers paging in a JSON body and
  required, format-checked fields.
- [E4 to E7: record shaping](./004-record-shaping.md) covers nested records, the first item, merged lists, and computed
  tree fields.
- [E8 and E9: the fallback read and bundled kinds](./005-fallback-and-bundled-kinds.md) covers the second read and the
  exception for embedded specs.
- [Schema and version impact](./006-schema-and-version-impact.md) says why schema version 1 stays and which module
  versions move.
- [Pull requests 6 to 15](./007-pull-requests-language.md) gives the language pull requests.
- [Pull requests 16 to 24](./008-pull-requests-ports-and-release.md) gives the ports, conformance, plugin, profiles, and
  release.
- [Outline counts and spec size](./009-outline-counts-and-spec-size.md) checks the 65 and the 66 in code and estimates
  whether one Outline spec fits the caps.
- [Kiwi probe](./010-kiwi-probe.md) lists what E1 to E9 cover for Kiwi, the gaps, and the stop-rule verdict.
- [Jira probe](./011-jira-probe.md) does the same for Jira.
