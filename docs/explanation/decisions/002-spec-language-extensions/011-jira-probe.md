---
title: Jira probe against the nine extensions
description: Which Jira operations E1 to E9 cover, the gaps they leave, and the verdict for the Jira port.
category: explanation
tags:
  - explanation
  - decision
  - planned
---

# Jira probe against the nine extensions

This page belongs to [ADR 002](../002-spec-language-extensions.md). It is a read-only study of
`libs/retriever-connector-jira`. It invents no extension. Each gap is an open item, with an id for later reference.

## What Jira is

- Jira Cloud has five read operations and no write path. The adapter calls five fixed requests, and one of them is a
  POST that reads (`/search/jql`).
- It signs in with HTTP Basic: the account e-mail and an API token. The spec already has `auth.type: basic` with
  `identity_env`. There is no session, no token refresh, no XML, and no multipart upload.

## Coverage by operation family

**`jira.whoami`** is a GET of three fields.

- Covered. There is no gap.

**`jira.projects.list`** is a GET with `startAt`, `maxResults`, and a `query`.

- Covered in part: a query-paged read with `max_pages: 1` and a continuation.
- Open: J1, the "more" flag comes from `isLast`, where the engine stops at a short page. J2, a reply with more items
  than the limit is cut to the limit.

**`jira.issues.search`** is a POST with the JQL, a fixed `fields` list, and a `nextPageToken`.

- Covered: E1 for the POST read, E2 for the cursor in the body, E3, and plain pointers for the nested status, type,
  priority, and assignee names.
- Open: J1 and J2 again, and the `truncated` flag that follows. J3, `projectKey` taken from the issue key. J4, a
  timestamp rewritten as RFC 3339 in UTC. J5, a title cut to a fixed number of characters on one line.

**`jira.issues.get`** returns one issue as a document.

- Covered in part: a GET with a constant `fields` query, and the description through `markdown-from-adf` (pull request
  18).
- Open: J0, a path parameter of the catalog type `IssueKey`, where the spec allows `ID`, `UUID`, and `ShareID` only.
  J6, the Markdown is assembled from many fields: a title, a metadata list, labels, a due date, a project line, and the
  description. That is not a converter of one field. J7, the reply key must equal the request key. J8, the document ID,
  the collection ID, and a URL (`/browse/<key>` on the endpoint origin) are built from the key.

**`jira.comments.list`** is a GET with an ADF body for each comment.

- Covered in part: the converter on a field of each list item, if pull request 18 serves it there.
- Open: J0, J1, J2, and J4.

## Verdict

Jira has no stop rule in decision D9. Pull request 20 says that an operation the nine features cannot express goes to
a new ADR.

- `jira.whoami` can port now.
- `jira.projects.list`, `jira.issues.search`, and `jira.comments.list` can port once the owner decides J0 to J5. Each is
  a small step, such as a converter for timestamps. Each would be a tenth feature, so each needs an ADR (decision D10).
- `jira.issues.get` cannot port with E1 to E9. J6 to J8 need a new ADR.

So the Jira port waits for a new ADR on J0 to J8. If the owner does not want it, Jira stays Go code, as Kiwi does.
`retriever-connector-jira` leaves the graph only in pull request 20, and only after that decision.
