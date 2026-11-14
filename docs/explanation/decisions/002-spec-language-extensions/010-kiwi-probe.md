---
title: Kiwi probe against the nine extensions
description: Which Kiwi operations E1 to E9 cover, the gaps they leave, and the verdict on the Kiwi stop rule.
category: explanation
tags:
  - explanation
  - decision
  - planned
---

# Kiwi probe against the nine extensions

This page belongs to [ADR 002](../002-spec-language-extensions.md). It is a read-only study of
`libs/retriever-connector-kiwi`. It invents no extension. Each gap is an open item, with an id for later reference.

## What Kiwi is

- Kiwi speaks JSON-RPC 2.0. Every call is a POST to one path, with a body of `jsonrpc`, `method`, named `params`, and an
  `id`. The key travels as a bearer token. There is no session, no token refresh, no XML, and no multipart upload.
- The catalog holds six operations: three reads (`kiwi.products.list`, `kiwi.products.tree`, `kiwi.documents.get`) and
  three writes (create a case, update a case, add a case to a plan). The adapter allows ten RPC methods.

## Coverage by operation family

**`kiwi.products.list`** is one `Product.filter` call.

- Covered: E1 (a POST read with constant `jsonrpc` and `method`), E3, and a reply pointer under `/result`.
- Open: K1, an error object inside an HTTP 200 reply; the engine maps HTTP statuses only. K2, the output ID
  `product-<n>`, which is built from a number.

**`kiwi.products.tree`** reads the plans of a product, then the cases of each plan, then a sort-key table for each plan.

- Not covered. K3: the read takes one request per plan and per sort-key table, and builds the tree from parent links. E7
  flattens a nested array in one reply and never joins several replies.

**`kiwi.documents.get`** returns a plan or a case as Markdown.

- Not covered. K4: a plan takes three calls, and a case takes a case read plus three lookup tables, which turn category,
  priority, and status IDs into names. The Markdown is assembled from many fields. E8 allows one second read, and a
  converter acts on one field.
- Open: K2, and K5, the input `plan-<n>` or `case-<n>`, which the adapter parses to a number.

**The three writes** build a values object, and resolve category, priority, and status names first.

- Covered: E1, the existing `resolve` for names, and the unknown-outcome rule. The lookups are POST reads, so `resolve`
  depends on E1.
- Open: K2 and K5 for IDs in and out. K6, the URL of the new case, built from the catalog endpoint. K7, update first
  reads the case and refuses a missing one, which is a read before a write.

**Across the adapter.**

- K8: calls go one at a time with a pause after each. The engine has no pacing.
- K9: the reply checks that the JSON-RPC `id` equals the request `id`.
- K10: text is unescaped one level of HTML, then stripped of invisible characters. The engine strips control
  characters only.

## Verdict

The Kiwi stop rule of decision D9 is met. Two of three reads cannot be written with E1 to E9 and declared lookups: K3
and K4 are structural, not small. All six operations also need prefixed IDs (K2, K5).

So Kiwi stays Go code until schema version 2, and the owner is told by this page. Pull request 19 becomes a decision
gate. It retires nothing, and `retriever-connector-kiwi` stays in the graph. Moving Kiwi would need a new ADR that names
K1 to K10 one by one.
