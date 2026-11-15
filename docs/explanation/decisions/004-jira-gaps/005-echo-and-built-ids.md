---
title: "E23 and E24: the key echo and the built IDs"
description: The reply-key check, and document and collection IDs with browse URLs built from the key.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E23 and E24: the key echo and the built IDs

This page belongs to [ADR 004](../004-jira-gaps.md). Pull request 35 built E23 and E24 as described here: the
key echo with its exact-equality check, and the built IDs with their shared issue-key check. E22 stays planned on
its own page. E23 and E24 cover J7, the reply key that must equal the request key in `issues.get`; J3, the
`projectKey` taken from the issue key; and J8, the document ID, the collection ID, and the `/browse/<key>` URL
built from the key.

## E23: the key echo

```yaml
response:
  echo: { param: issueKey, pointer: /key } # new; the reply value must equal the request arg
```

- **Rule.** A single-object read may declare one echo. After mapping, the text at the pointer must equal the
  request argument exactly, or the read fails with `unexpected_response` naming the pointer path, never the
  values. The load refuses an echo on a list, tree, or merge response, and a pointer to a non-text value
  fails the same way.
- **L26. A reply is accepted for the wrong key.** Control: the exact-equality check above, run by the engine
  on every call. Proof: `TestKeyEchoMismatchIsUnexpected`, `TestKeyEchoRefusalNamesPath`, and
  `TestKeyEchoOnListIsRefused`.
- **Cannot do.** Echoing a list, echoing two params, or comparing past equality.

## E24: the built IDs

```yaml
fields:
  projectKey: { project_key: /key } # new; the text before the last hyphen
  id: { issue_id: /key } # new; issue-<KEY-123>
  collectionId: { collection_id: /key } # new; project-<KEY>
  url: { browse_url: /key } # new; <endpoint-origin>/browse/<KEY-123>
```

- **Rule.** All four read the text at one pointer and share one closed issue-key check: the value must pass
  the E19 shape or the read fails with `unexpected_response` naming the field, never the value.
  `project_key` answers the text before the last hyphen; `issue_id` prefixes it with `issue-`;
  `collection_id` prefixes the project part with `project-`; `browse_url` joins the endpoint origin of the
  operation with `/browse/` and the key. The base is always the endpoint the source pins, and any URL text
  the reply carries is ignored. J3 rides this shared check, so one rule serves both directions of the key.
- **L27. A reply URL or ID text reaches the output, or a built ID misnames.** Control: the fixed prefixes,
  the fixed path, and the ignore-reply-text rule above. Proof: `TestBrowseUrlIgnoresReplyUrl` (plants an
  attacker URL in the reply and finds the endpoint-origin URL in the output), `TestBuiltIdRefusesBadKey`,
  and `TestProjectKeyTakesPrefix`.
- **Cannot do.** Another prefix, another path, a query string, a base from reply data, or building from a
  non-key.
