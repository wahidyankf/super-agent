---
title: "Pull requests 25 to 32: the Kiwi extensions, the port, and the retirement"
description: The language pull requests for K1 to K10, then the Kiwi port and the module retirement.
category: explanation
tags:
  - explanation
  - delivery
  - planned
---

# Pull requests 25 to 32: the Kiwi extensions, the port, and the retirement

This page is the migration order of [ADR 003](../003-kiwi-gaps.md). The shared gates of
[ADR 002](../002-spec-language-extensions/007-pull-requests-language.md) apply to every pull request here: the
pull-request surface, `lint`, `typecheck`, `test:unit` at the 99 percent floor, and `test:integration` for each Go
module touched, plus a changelog entry and a docs update in the same change. The spec modules move first, then the
port, then the retirement. Each language pull request follows the five-part shape of ADR 002: failing tests first,
schema keyword plus Go rule, engine behaviour with stub support, review text, and the fields page moved to as built.
The schema stays at version 1; `retriever-spec` and `retriever-engine` take one minor bump in pull request 25.

## Pull requests at a glance

| #   | Pull request                              | Builds             | Retires    |
| --- | ----------------------------------------- | ------------------ | ---------- |
| 25  | E18 envelope and E10 error object         | K9, K1             | nothing    |
| 26  | E11 prefixed IDs and E15 built URLs       | K2, K5, K6         | nothing    |
| 27  | E17 pacing and the `clean-text` converter | K8, K10            | nothing    |
| 28  | E16 write guard                           | K7                 | nothing    |
| 29  | E12 fan-out trees                         | K3                 | nothing    |
| 30  | E13 assembly and E14 Markdown             | K4                 | nothing    |
| 31  | Port Kiwi onto a bundled spec             | the six operations | nothing    |
| 32  | Retire `retriever-connector-kiwi`         | the no-import test | the module |

## 25. E18 envelope and E10 error object

Scope: the fixed JSON-RPC body, the sequential id, the echo check, the `error_object` flag with the fixed code and
word table, and the closed `not_found` names. Gate: swapped reply ids fail; "login required" stays key rejected; a
secret planted in a message reaches no error or log.

## 26. E11 prefixed IDs and E15 built URLs

Scope: the shared canonical-number check, the exact prefix, both directions, and the endpoint-origin URL rule that
ignores reply URLs. Gate: `product-01` is refused naming the path; a planted reply URL never reaches the output.

## 27. E17 pacing and clean-text

Scope: the per-source delay with the 1000 millisecond ceiling and serialization, and the fixed unescape-and-strip
converter. Gate: a fake clock observes the spacing; a zero-width space and a bidi override reach no output.

## 28. E16 write guard

Scope: the one guard read, params from the write's args only, and the declared not-found with zero writes on a miss.
Gate: a missing case sends no write; the race stays closed by the unknown-outcome rule on the reply.

## 29. E12 fan-out trees

Scope: same-spec child reads with the 200-call cap, the shared deadline, map-shaped attach with ordering, and the
parent-link assembly with the list-once rule. Gate: 201 plans are refused before any case is read; a parent loop
lists each plan once.

## 30. E13 assembly and E14 Markdown

Scope: at most four same-spec read steps with all-or-nothing merge, the match-by-id pick, and the declarative layout
over cleaned fields. The writes keep riding `resolve`. Gate: any step failure fails the read with no partial record;
planted markup stays literal text.

## 31. Port Kiwi

A bundled `kiwi.yml` serves the six catalog operations with `catalog_id` mapping per E9, reusing E1 and `resolve`
where the probe found them covered. The unchanged Kiwi read, write, redaction, and transport scenarios run on the
engine and pass without an edit. Gate: `TestBundledSpecCoversEveryCatalogRow` for Kiwi, and no scenario edited.

## 32. Retire the module

`retriever-connector-kiwi` leaves the dependency graph. An architecture test fails on any new import of it. The
CHANGELOG records each breaking change of this ADR at port time: the removed module and any refined error mapping.
Exit gate for the ADR: the shared gates, every named test passing, `TestExistingSpecsStillValidate`,
`TestSchemaKeywordLock` extended to E10-E18 and `clean-text`, and no Go connector serving Kiwi.
