---
title: "E21: the Jira text converters"
description: Closed converters for Jira timestamps, due dates, and one-line titles.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E21: the Jira text converters

This page belongs to [ADR 004](../004-jira-gaps.md). E21 is built by pull request 34. It covers J4, the
`timestampText` and `dueText` rewrites of `markdown.go`, and J5, the `titleText` cut. Each converter is a pure
function of one reply value, with the algorithm moved from the Go code and reviewed once.

```yaml
fields:
  updated: { pointer: /fields/updated, convert: jira-time } # new converter name
  due: { pointer: /fields/duedate, convert: jira-date } # new converter name
  summary: { pointer: /fields/summary, convert: jira-title } # new converter name
```

- **Rule.** `jira-time` parses the fixed Jira layout (`2006-01-02T15:04:05.000-0700`) and answers the UTC
  instant as RFC 3339. `jira-date` answers a `YYYY-MM-DD` value unchanged when it parses as a calendar day.
  Any other value, including a missing one, maps to the empty text, exactly as the Go code answers today: the
  field stays present and empty, so the row is kept and the gap is visible. `jira-title` turns every line
  break and tab into one space, strips the fixed hidden-character set, and keeps the first 255 runes. The
  names say the fixed reply layout they read, as `markdown-from-adf` does; no spec field changes them.
- **Why empty instead of loud.** The scenarios pin the empty text for a broken date. Failing the row would
  move behaviour the port must keep. The empty field is observable output, not a silent drop.
- **L24. A converter hides evidence, or passes a bad value through as text.** Control: the fixed layouts and
  the fixed strip set above, with parity against the moved Go tests. Proof: `TestJiraTimeMapsBadLayoutToEmpty`
  (plants a shifted layout and finds the empty text), `TestJiraTitleKeeps255Runes`,
  `TestJiraTitleStripsInvisible` (plants a zero-width space and a bidi override and finds neither), and
  `TestJiraDateRefusesBareYear`.
- **Cannot do.** Another layout, a second parse attempt, a longer title, or cleaning after mapping.
