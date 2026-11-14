---
title: "Pull requests 6 to 15: the spec language extensions"
description: The ten pull requests that add the nine extensions to the spec language before any connector is ported.
category: explanation
tags:
  - explanation
  - delivery
  - planned
---

# Pull requests 6 to 15: the spec language extensions

This page replaces pull request 6 of [ADR 001](../001-user-defined-connectors-and-profiles/006-pull-requests-engine.md)
for [ADR 002](../002-spec-language-extensions.md). Pull requests 1 to 5 of ADR 001 and the new pull request 6
are merged. The [ports and the rest](./008-pull-requests-ports-and-release.md) come next. Each pull request leaves
`main` working.

## Gates that every pull request shares

- The standard gates of [ADR 001](../001-user-defined-connectors-and-profiles/006-pull-requests-engine.md): the
  pull-request surface, and `lint`, `typecheck`, `test:unit` at the 99 percent floor, and `test:integration` for each Go
  module touched.
- A changelog entry, and a docs update in the same change.

## The shape of a language pull request

Pull requests 7 to 15 each add one feature, in five parts, and in this order:

1. The tests named on the feature page, each seen to fail first.
2. The schema keyword in `connector-spec-v1.schema.json` and the Go rule in `retriever-spec`.
3. The behaviour in `retriever-engine`, and support in the replay stub.
4. The review text of `retriever-spec`, so a reviewer sees the feature.
5. The feature on [connector spec fields](../../../reference/connector-spec-fields.md) moved from planned to as built.

Exit gate for each: the shared gates, every named test passing, `TestExistingSpecsStillValidate` and
`TestSchemaKeywordLock` passing, and no change to any Go connector.

## 6. The golden (done, #25)

Merged. Test-only. It landed `outline-read-tools.golden`, which pins the name, description, input schema, and hints of
the 65 Outline read tools, and `TestOutlineReadToolsMatchTheGoBackedGolden`. It also added the "Limits for the Outline
reads" note to the spec fields page, which describes the language before this ADR.

The written probe of Kiwi and Jira that this pull request was to carry is part of this ADR instead:
[Kiwi](./010-kiwi-probe.md) and [Jira](./011-jira-probe.md). The probes found gaps, so the ports in pull requests 19 and
20 are gated. The Outline path, pull requests 7 to 18, is not affected.

## 7. E1: POST reads (done)

Scope: the marker, the method matrix, the body rules, the review block, and the hint rule. This pull request bumps
`retriever-spec` and `retriever-engine` to `v0.5.0`. Gate: the owner confirms the hint rule for unpinned user specs,
because it decides S11 again.

## 8. E2: body paging

Scope: `in: body` for offset and cursor paging. It settles the cap behaviour of `collections.list` (see E2). Gate: a
replay run of 25 pages stops at the cap of 20.

## 9. E3: required and validated fields

Scope: `required` and `format` on a reply field. Gate: a request with a missing argument leaves no request at the stub.

## 10. E4: nested record with an allow-list

Scope: `object`. It first checks that the renderers, the MCP adapter, and the JSON output handle a nested record, and
bumps the contract if not. Gate: a planted e-mail reaches no output, error, or log.

## 11. E5: first item or not found

Scope: `first`. Gate: an empty list gives `not_found`, and no partial record.

## 12. E6: merged lists with constant labels

Scope: `merge`. Gate: the label beats a reply field of the same name.

## 13. E7: computed tree fields

Scope: `tree`, with `parentId`, `depth`, and the cap of 64. It also raises the decode cap
of a tree reply to 136 levels. Gate: a tree nested to 70 levels keeps 65 levels of nodes, and any other reply nested
past 64 levels still fails.

## 14. E8: the fallback read

Scope: `fallback`, with the triggers `empty` and `status`. Gate: a chain of two fallbacks is refused at load.

## 15. E9: bundled kinds and the ID mapping

Scope: `ParseBundled`, `catalog_id`, and the embed loader, with no spec yet. Gate: a copy of a bundled spec in the user
folder is refused, and an architecture test finds one caller of `ParseBundled`.
