---
title: "E1: POST reads"
description: The strict rule that lets a spec declare a POST as a read, and why that is safe enough.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E1: POST reads

This page belongs to [ADR 002](../002-spec-language-extensions.md). It decides threat S11 again. Everything here is
planned.

## What it adds

Today a read takes `GET` or `HEAD`. Outline, like many JSON APIs, reads with `POST`. E1 lets a read use `POST` under a
strict rule, and nothing else about the read changes.

```yaml
- id: outline.documents.list
  access: read
  method: POST
  read_over_post: true
  path: /documents.list
  params:
    - { name: collection_id, type: ID, in: body, required: false }
  body:
    collectionId: "{{collection_id}}"
```

## What makes a POST a read

All five conditions hold, or the spec is refused with `retriever.connector.spec_invalid`:

1. The operation has `access: read` and `read_over_post: true`. The marker has no other use.
2. The body is a JSON object whose values are constants or whole `{{name}}` slots of declared body parameters.
3. The engine sets the content type to `application/json`. A spec may not set `Content-Type`.
4. The operation has no `resolve`. It claims no side effect, because the spec has no field for one.
5. The rendered body is at most 64 KiB and nests at most 8 levels. These limits are proposed, and a spec may only lower
   them.

## Re-deciding S11

ADR 001 said the engine cannot know whether `access` is true. That stays true for a POST. What changes is the cost of a
lie. An MCP client may run a read-only tool without asking, so a spec that labels a write as a read could turn a
trusted call into a change. The controls are these, and none of them guards an action:

- **Method matrix.** `GET` and `HEAD` take `read`. `POST` takes `read` only with the marker, and `write` or `admin`
  without it. `PUT`, `PATCH`, and `DELETE` take `write` or `admin` only. Any other pair fails validation.
- **Review text.** `connector validate` lists every POST read in its own block, with its path, so a reviewer sees each
  one.
- **Conformance check.** For a bundled spec, the access label must equal the effect of its catalog row. A user spec
  cannot be checked this way.
- **Honest hints.** A bundled POST read reports read-only MCP hints, as the golden shows. A POST read in a user spec
  reports them only when its catalog source pins the spec with `spec_sha256`, because the pin says the user reviewed
  those bytes. Otherwise its hints are those of a write. A hint is a label, never a guard. This rule is a proposal that
  the owner can still veto at review.
- **Fail closed.** The engine re-checks the matrix in `New`. A model that breaks it is refused before any request.

What stays open: a pinned spec can still lie, and the token's own access remains the only guard (D1).

## Threats

**L1. A POST labelled read changes data.** Control: the matrix, the marker, the review block, and the hint rule above.
Proof: `TestAccessMethodMatrix`, `TestPostReadNeedsMarker`, `TestReviewListsPostReads`, and
`TestUnpinnedUserPostReadHintsAreNotReadOnly`.

**L2. A body carries more than the declared slots.** Control: conditions 2 and 3. A value fills a whole JSON value, so
it never splices into text (S1). Proof: `TestPostReadBodyOnlyFromDeclaredSlots` sends a value that looks like a slot and
finds it literal. `TestPostReadContentTypeFixed` fails a spec that sets `Content-Type`.

A BDD scenario also sends one POST read to the replay stub and checks the method, the body, and a single request.

## What E1 cannot do

Send a form, multipart, or XML body. Take a body from anything but declared slots. Send a body with `GET`. Prove that
the server has no side effect.
