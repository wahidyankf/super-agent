---
title: ADR 004 companion pages
description: The threat model delta, the nine Jira gap decisions, and the pull-request sequence of ADR 004.
category: explanation
tags:
  - index
  - explanation
  - decision
  - planned
---

# ADR 004 companion pages

These pages hold the detail of [ADR 004](../004-jira-gaps.md). Everything on them is planned.

## Directory Map

- [Threat model delta](./001-threat-model-delta.md) lists what the Jira extensions change in the assets,
  boundaries, and entry points, and the new threats L22 to L28.
- [E19 and E20: the issue key and the page end](./002-issue-key-and-page-end.md) covers path parameters in
  the `IssueKey` shape (J0), the `isLast` stop rule (J1), and the cut of over-limit pages (J2).
- [E21: the Jira text converters](./003-converters.md) covers timestamps rewritten as RFC 3339 in UTC (J4)
  and titles cut to one line of 255 runes (J5).
- [E22: the issue document layout](./004-document-layout.md) covers the Markdown assembled from many fields
  (J6).
- [E23 and E24: the key echo and the built IDs](./005-echo-and-built-ids.md) covers the reply key check
  (J7), the project key (J3), and the built document and collection IDs with browse URLs (J8).
- [Pull requests 33 to 39](./006-pull-requests.md) gives the language pull requests, the Jira port, and the
  retirement of `retriever-connector-jira`.
