---
title: "E2 and E3: body paging and validated fields"
description: Paging with offset and limit in a JSON body, and required, format-checked fields on a request or reply.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E2 and E3: body paging and validated fields

This page belongs to [ADR 002](../002-spec-language-extensions.md). E2 is built by pull request 8. E3 is still
planned.

## E2: body paging

Outline pages with `offset` and `limit` in the JSON body. Before pull request 8, a paging rule named query
parameters only.

```yaml
pagination:
  style: offset # or cursor
  in: body # new; the default stays query
  limit_param: limit
  offset_param: offset
  max_pages: 20
```

- **Rule.** With `in: body`, every parameter that the paging rule names must be a declared body parameter of the same
  operation. A rule cannot mix locations. A cursor in the body also names `next_pointer`.
- **Limits.** The page cap is the same as for a query: 1 to 20, and a spec may only lower it. The offset is a
  non-negative integer below 2^31. The engine advances it by the number of items in the page, and stops at the first
  short page, as the offset read does today. At the cap it returns the pages it has, with a continuation.
- **Decided.** The Go `collections.list` keeps failing at the cap with a page-cap error, and the engine keeps
  returning the pages it has with a continuation. The gateway call returns a list or an error and has no channel for a
  continuation, so failing is its only honest option, while a spec read returns a page that names its continuation
  (`HasMore` and `NextOffset`), so the caller continues. Pull request 8 records this, and the owner is told.
- **Needs.** E1, because a body-paged read is a POST read.

**L3. Paging in a body runs past the cap or never advances.** Control: the cap and the no-progress stop above. This
extends S3. Proof: `TestBodyOffsetPagingStopsAtMaxPages`, `TestBodyPagingStopsWithoutProgress`,
`TestBodyCursorPaging`, and `TestPagingRuleRefusesMixedLocations`.

E2 cannot page by a header, grow the page size, or read two pages in parallel.

## E3: required and validated fields

The request side exists today: a missing required argument fails with `missing_argument` before any request, and a bad
ID fails with `invalid_id`. The reply side is the gap. The Go connectors refuse a reply that holds no ID: `requiredID`
in the Outline `decode.go` returns `unexpected_response` when the field is missing or its ID check fails, and 22 call
sites use it. A spec maps a missing reply field to null today. E3 closes that gap.

```yaml
response:
  fields:
    id:
      pointer: /data/id
      required: true # new: a missing, null, or empty value fails the read
      format: id # new: a closed name, checked on the value
```

- **Rule.** `required: true` on a reply field makes a missing, null, or empty value an `unexpected_response`. No partial
  record is returned. A `format` on a reply field is one of the catalog's ID kinds: `id`, `uuid`, or `share-id`. It is a
  closed name, not a pattern. A request parameter keeps its catalog `type`, which already checks the same kinds.
- **Why no free pattern.** A pattern from a spec is untrusted input and is costly to review. A closed name is reviewed
  once, in the engine.
- **Limits.** Failure names the field path and never the value (R1).

**L4. A bad ID is sent, or a reply with no ID is trusted.** Control: the checks above, run before any request on the
request side and before mapping on the reply side. Proof: `TestRequiredParamIsRefusedBeforeAnyRequest`, where the stub
counts zero requests. `TestRequiredReplyFieldMissingIsUnexpected`, `TestReplyFormatMismatchIsUnexpected`,
`TestFormatNamesAreClosed`, and `TestRefusalNamesPathNotValue` cover the reply side.

E3 cannot check a value against a pattern, compare two fields, or require one field only when another is present.
