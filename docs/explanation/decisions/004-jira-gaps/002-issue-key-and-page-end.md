---
title: "E19 and E20: the issue key and the page end"
description: The closed IssueKey path parameter, the isLast stop rule, and the cut of over-limit pages.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E19 and E20: the issue key and the page end

This page belongs to [ADR 004](../004-jira-gaps.md). E19 and E20 are built by pull request 33. They cover J0,
the `IssueKey` path parameters of `issues.get` and `comments.list`; J1, the `isLast` flag of `projects.list`
and `issues.search`; and J2, the cut of a page that holds more items than the limit.

## E19: the issue key

A Jira issue key is an upper-case project key, a hyphen, and 1 to 9 digits with no leading zero. The spec
allows only `ID`, `UUID`, and `ShareID` path parameters, and `ID` accepts lowercase. E19 adds one closed
parameter type.

```yaml
params:
  - { name: issueKey, type: IssueKey, in: path, required: true } # new type; path only
```

- **Rule.** The schema gains `IssueKey` in the parameter type enum and in the path-parameter allow-list. The
  engine binds it through the existing `KindIssueKey` validator, so no new check is written: a value outside
  the shape fails before any request with `invalid_document_id`, naming the parameter path and never the
  value.
- **L22. A bad issue key is sent, or a lowercase key is accepted.** Control: the shared strict check above,
  which refuses lowercase, missing hyphens, leading zeros, and traversal text. Proof:
  `TestIssueKeyLowercaseIsInvalid`, `TestIssueKeyTraversalIsInvalid`, `TestIssueKeyRefusalNamesPath`, and
  `TestIssueKeyCanonicalAccepted`.
- **Cannot do.** Another shape, a second key type on one parameter, case-insensitive matching, or building a
  key from reply data.

## E20: the page end

The engine stops an offset walk at the first short page and returns every item a page holds. Jira decides
with `isLast` instead, and cuts a page that holds more than the limit, marking more. E20 adds two closed
pagination keys and one computed response flag.

```yaml
pagination:
  style: offset
  limit_param: limit
  offset_param: offset
  max_pages: 1
  last_pointer: /isLast # new; the reply flag that decides last
  cut_to_limit: limit # new; the parameter whose value cuts the page
response:
  items: /issues
  truncated: { token: /nextPageToken, last: /isLast } # new; per-item flag, search only
```

- **Rule.** When `last_pointer` is declared, the reply must hold a bool there or the read fails with
  `unexpected_response` naming the pointer path; more is the negation of that flag, beating the short-page
  rule. When `cut_to_limit` names a parameter, a page holding more items than that value keeps the first
  items and sets more. The named parameter must exist and be of Limit or Int kind. The `truncated` block is
  allowed only with a cursor rule that declares `last_pointer`: each item gains `truncated`, which is true
  exactly when the token is present and non-empty and last is false. A missing token counts as empty; a
  missing last counts as false, as the Go code reads them. A non-bool last or a non-string token is
  `unexpected_response` naming the path.
- **Why not the short-page rule.** A full page can still be last, and Jira says so with `isLast`. Trusting
  the count alone claims more pages that do not exist, or misses the cut that keeps the next page aligned.
- **L23. A short page is misread as last, or an over-limit reply passes whole.** Control: the shape-checked
  flag and the first-N cut above. Proof: `TestLastPointerFullPageIsLast`,
  `TestLastPointerMissingIsUnexpected`, `TestCutToLimitKeepsFirstN`,
  `TestTruncatedNeedsTokenAndNotLast`, and `TestTruncatedMissingTokenIsFalse`.
- **Cannot do.** A last flag from args, a cut to a reply number, a truncated flag without a last rule, or a
  second last pointer.
