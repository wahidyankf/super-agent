---
title: "Pull requests 16 to 24: ports, conformance, plugin, profiles, and release"
description: The pull requests that port the three connectors, then add conformance, plugin, profiles, and v0.3.0.
category: explanation
tags:
  - explanation
  - delivery
  - planned
---

# Pull requests 16 to 24: ports, conformance, plugin, profiles, and release

This page continues the [language pull requests](./007-pull-requests-language.md). The shared gates listed there apply
here too. Every port runs the existing behaviour scenarios without an edit, so the scenarios prove that nothing moved. A
Go connector module leaves the dependency graph only in the pull request that ports its last operation.

## Pull requests at a glance

| #   | Pull request                                 | Retires                       |
| --- | -------------------------------------------- | ----------------------------- |
| 16  | Port Outline reads                           | nothing                       |
| 17  | Port Outline writes and admin                | `retriever-connector-outline` |
| 18  | The `markdown-from-adf` converter            | nothing                       |
| 19  | Kiwi decision gate (stop rule met)           | nothing                       |
| 20  | Port Jira, after a new ADR                   | `retriever-connector-jira`    |
| 21  | Conformance suite and the connector commands | nothing                       |
| 22  | The plugin host                              | nothing                       |
| 23  | Profiles                                     | nothing                       |
| 24  | The `v0.3.0` pre-release                     | nothing                       |

## 16. Port Outline reads

A bundled `outline.yml` serves the 66 catalog reads, each with a `catalog_id` and an example, and the fallback target
`documents.export` (see [the counts](./009-outline-counts-and-spec-size.md)). A composite builder routes each operation
to the spec or to the Go gateway until pull request 17.

- Given the unchanged Outline read, redaction, and transport scenarios, when they run on the engine, then every one
  passes.
- Given `outline-read-tools.golden`, when the MCP server lists the 65 read tools, then the text is byte-identical.
- Gate: `TestBundledSpecCoversEveryCatalogRow` for reads. The pull request measures the spec size first. The estimate
  is near the 256 KiB cap. If Outline does not fit, the work stops and the owner decides (open question on that page).

## 17. Port Outline writes and admin

The bundled spec gains the write, destructive, and admin operations. The composite builder and
`retriever-connector-outline` leave the graph. Docs: the modules, tags, and layering pages.

- Given the unchanged write and destructive scenarios, then every one passes.
- Given a write whose outcome is unknown, then the code is `write_outcome_unknown` and the write is never retried.
- Gate: the architecture test shows no import of the retired module.

## 18. The `markdown-from-adf` converter

The engine serves the closed converter `markdown-from-adf` as a pure function, so it no longer fails an operation that
names it. Only Jira needs it, so it waits for the decision named under pull request 20. The Jira module keeps a copy of
the function until it retires.

- Given a document tree nested past the depth cap, then the output ends with the truncation marker and nothing panics.
- Gate: the Jira converter tests pass unchanged against the moved function.

## 19. Kiwi decision gate

The [Kiwi probe](./010-kiwi-probe.md) meets the stop rule of decision D9: Kiwi stays Go code until schema version 2.
This pull request records that in the docs and retires nothing. A port needs a new ADR on gaps K1 to K10 first. If one
is accepted, the port keeps these scenarios:

- Given the unchanged Kiwi read and write scenarios, then every one passes.
- Given a category name that matches no entry, then the write fails with `unresolved_name`, and nothing is sent.

## 20. Port Jira

It waits for a new ADR on the gaps J0 to J8 of the [Jira probe](./011-jira-probe.md). Then the bundled `jira.yml` uses
the converter of pull request 18, and `retriever-connector-jira` leaves the graph.

- Given the unchanged Jira scenarios, then every one passes.
- Stop rule: an operation that the accepted features cannot express stays Go code and goes to the owner, never to an ad
  hoc feature.

## 21 to 23. Conformance, plugin, and profiles

These keep the scope, scenarios, and proofs of pull requests 10, 11, and 12 in
[ADR 001](../001-user-defined-connectors-and-profiles/007-pull-requests-ports-and-profiles.md). The conformance suite
(21) also runs over the three bundled specs and checks every `catalog_id` (L10). The plugin host (22) and profiles (23)
change nothing in the spec language.

## 24. The `v0.3.0` pre-release

Version bump, the changelog moved from Unreleased, and the release notes, as a pre-release marked ALPHA. Gate: the
release gate, the public-safety preflight, and a changelog that lists each **Breaking** change of this ADR.
