---
title: "Pull requests 33 to 39: the Jira extensions, the port, and the retirement"
description: The language pull requests for J0 to J8, then the Jira port and the module retirement.
category: explanation
tags:
  - explanation
  - delivery
  - planned
---

# Pull requests 33 to 39: the Jira extensions, the port, and the retirement

This page is the migration order of [ADR 004](../004-jira-gaps.md). The shared gates of
[ADR 002](../002-spec-language-extensions/007-pull-requests-language.md) apply to every pull request here: the
pull-request surface, `lint`, `typecheck`, `test:unit` at the 99 percent floor, and `test:integration` for each Go
module touched, plus a changelog entry and a docs update in the same change. The spec modules move first, then
the port, then the retirement. Each language pull request follows the five-part shape of ADR 002: failing tests
first, schema keyword plus Go rule, engine behaviour with stub support, review text, and the fields page moved to
as built. The schema stays at version 1; `retriever-spec` and `retriever-engine` take one minor bump in pull
request 33. The Jira port builds on the Kiwi language pull requests 25 to 30: E10 `not_found` names, E14
converters beside E21, and E17 pacing for the tree walk.

## Pull requests at a glance

| #   | Pull request                      | Builds              | Retires    |
| --- | --------------------------------- | ------------------- | ---------- |
| 33  | E19 issue key and E20 page end    | J0, J1, J2          | nothing    |
| 34  | E21 Jira text converters          | J4, J5              | nothing    |
| 35  | E23 key echo and E24 built IDs    | J7, J3, J8          | nothing    |
| 36  | E22 issue document layout         | J6                  | nothing    |
| 37  | E25 query key                     | the Jira wire names | nothing    |
| 38  | Port Jira onto a bundled spec     | the five operations | nothing    |
| 39  | Retire `retriever-connector-jira` | the no-import test  | the module |

## 33. E19 issue key and E20 page end

Scope: the `IssueKey` parameter type with the shared strict check, the `last_pointer` stop rule, the
`cut_to_limit` cut, and the computed `truncated` flag. Gate: a lowercase key is refused naming the path; a
full page with `isLast` true claims no further page; a planted token with no last flag marks truncated.

## 34. E21 Jira text converters

Scope: the fixed `jira-time`, `jira-date`, and `jira-title` algorithms moved from the Go code, reviewed once.
Gate: the moved Go parity tests pass unchanged; a bidi override reaches no output; a bad date maps to the
empty text with the field present.

## 35. E23 key echo and E24 built IDs

Scope: the exact-equality echo on single-object reads, and the four built keys sharing one issue-key check,
with endpoint-origin URLs only. Gate: a mismatched key fails naming the pointer path; a planted reply URL
never reaches the output.

## 36. E22 issue document layout

Scope: the title, the closed line kinds with defaults and skips, the moved escape algorithm, and the
description body with its fixed heading and empty text. Gate: planted markup stays literal text; empty
resolution, labels, parent, and due lines are absent; the defaults are byte-identical.

## 37. E25 query key

Scope: the closed optional `query_key` string on `in: query` params only, so the Jira port names CLI params `limit`
and `offset` while the wire carries `maxResults` and `startAt`. Gate: an alias on a non-query parameter is refused
naming the alias; an alias repeating the parameter name is refused; two query keys sharing one wire key are refused;
and the offset walk sends the alias while binding the catalog name.

## 38. Port Jira

A bundled `jira.yml` serves the five catalog operations with `catalog_id` mapping per E9, reusing E1, E2, and
`resolve` where the probe found them covered. `whoami` ports with no new feature. The collection walk rides
page-all, and the tree walk rides a second cursor read with E20 and E17. The unchanged Jira read, redaction,
and transport scenarios run on the engine and pass without an edit. Gate: `TestBundledSpecCoversEveryCatalogRow`
for Jira, and no scenario edited.

## 39. Retire the module

`retriever-connector-jira` leaves the dependency graph. An architecture test fails on any new import of it. The
CHANGELOG records each breaking change of this ADR at port time: the removed module and any refined error
mapping. Exit gate for the ADR: the shared gates, every named test passing, `TestExistingSpecsStillValidate`,
`TestSchemaKeywordLock` extended to E19-E25 and the three converters, and no Go connector serving Jira.
