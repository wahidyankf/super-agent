---
title: "E11 and E15: prefixed IDs and built URLs"
description: Canonical prefixed IDs in both directions, and document URLs built from the endpoint origin only.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E11 and E15: prefixed IDs and built URLs

This page belongs to [ADR 003](../003-kiwi-gaps.md). E11 is built by pull request 26 and covers K2, the `product-<n>`
output of `reader.go`, and K5, the `plan-<n>` and `case-<n>` parsing of `ids.go`. E15 is built with it and covers K6,
the endpoint-built URLs of `reader.go`.

## E11: prefixed IDs

A Kiwi ID is a prefix and a canonical number: `product-<n>`, `plan-<n>`, or `case-<n>`, with 1 to 9 digits and no
leading zero. The engine checks catalog ID kinds today (E3), but it can neither parse nor build a prefixed form. E11
adds one rule that works in both directions.

```yaml
params:
  - { name: collection_id, type: prefixed_id, prefix: product- } # request side, K5
response:
  fields:
    id: { pointer: /result/id, prefix: product- } # reply side, K2
```

- **Rule.** The canonical number is 1 to 999999999, in ASCII digits, with no leading zero. The check is shared by both
  directions. A request value outside the form fails with `invalid_id`, naming the parameter path and never the value.
  A reply number outside the range fails the read with `unexpected_response`. The prefix is a constant of lowercase
  letters and one trailing hyphen, at most 32 characters, matched exactly.
- **Why one feature.** K2 builds the text and K5 parses it, but the canonical form is one rule. A single shared check
  cannot drift between directions.
- **L13. A non-canonical ID is accepted, or two texts name one number.** Control: the shared canonical check and the
  exact prefix above. Proof: `TestPrefixedIdLeadingZeroIsInvalid`, `TestPrefixedIdBuildsCanonicalText`,
  `TestPrefixedIdRefusalNamesPath`, and `TestPrefixMustBeLowercaseHyphen`.
- **Cannot do.** Patterns, several prefixes on one field, case-insensitive matching, or building from a non-number.

## E15: built URLs

A Kiwi document URL is the catalog endpoint plus `/plan/<n>/` or `/case/<n>/`. A URL in a reply is never used. The
engine has no URL rule, so E15 adds one that builds from the endpoint only.

```yaml
response:
  fields:
    url: { doc_url: plan, id_pointer: /result/id } # segment constant, number from the reply
```

- **Rule.** The base is always the origin of the operation's own endpoint. The path is one constant segment, `plan`
  or `case`, then the canonical number from `id_pointer`, then a trailing slash. The reply contributes the number
  only; any URL text in the reply is ignored.
- **L16. A reply URL is trusted and reaches another host.** Control: the fixed base and the ignore-reply-text rule
  above. The base cannot name another host because it is the endpoint the source already pins. Proof:
  `TestReplyUrlIsIgnored` (plants an attacker URL in the reply and finds the endpoint-origin URL in the output) and
  `TestDocUrlNeedsCanonicalNumber`.
- **Cannot do.** Another base, a query string, a segment from reply data, or following a reply URL.
