---
title: "Pull requests 7 to 12: ports, conformance, plugin, and profiles"
description: The last six pull requests of ADR 001, each with its scope, acceptance scenarios, and extra gates.
category: explanation
tags:
  - explanation
  - delivery
  - planned
---

# Pull requests 7 to 12: ports, conformance, plugin, and profiles

This page continues [pull requests 1 to 6](./006-pull-requests-engine.md), and the standard gates listed there apply.
Each port runs the existing scenarios unchanged, so the behaviour specs prove that nothing moved.

## 7. Port Outline writes and admin, and retire the Go module

The bundled spec gains the write, destructive, and admin operations. The composite builder and the
`retriever-connector-outline` module leave the graph. Docs: modules, tags, and layering pages. Extra gate: the
architecture test shows no import of the retired module.

- Given the unchanged Outline write and destructive scenarios, when they run against the engine, then every one passes.
- Given a write whose outcome is unknown, when it runs through the engine, then it reports the same error code as the
  old module and never retries.

## 8. Port Kiwi

The bundled `kiwi.yml` uses declared lookups for categories, priorities, and statuses. The Kiwi module retires.

- Given the unchanged Kiwi read and write scenarios, when they run against the engine, then every one passes.
- Given a category name that matches no entry, when a test case is created, then the engine reports the unresolved-name
  error and sends no write.
- Stop rule: if lookups cannot express a Kiwi scenario, a new ADR records why, and Kiwi stays Go code.

## 9. Port Jira

A closed converter, `markdown-from-adf`, joins the engine as a pure function moved from the Jira module. The bundled
`jira.yml` uses it. The Jira module retires.

- Given the unchanged Jira scenarios, when they run against the engine, then every one passes.
- Given a document tree nested past the depth cap, when the converter runs, then the output ends with the truncation
  marker and no panic occurs.

## 10. Conformance suite and the connector commands

`retriever-conformance` checks any spec: the schema is valid, every operation has examples, the examples replay through
the engine and match their expected fields, and the CLI, MCP, and README text derived from the spec equals the stored
text. `connector validate <spec>` and `connector test <spec>` run it. CI runs it over the three bundled specs. Proves
R3 and C3.

- Given a spec with an operation that has no example, when `connector validate` runs, then it exits `2` and names the
  operation.
- Given a spec whose example expects a field that the mapping does not produce, when `connector test` runs, then it
  exits `2` and names the example and the field.
- Given a credential value planted in a canned reply, when the report prints, then it shows `[REDACTED]`.
- Given a bundled spec with a host outside the three example domains, when the suite runs, then it fails.

## 11. The plugin host

`retriever-plugin` starts a pinned plugin and speaks JSON-RPC 2.0. The exit-code page changes: it says the tool starts a
child process only for a plugin, and that a failed start exits `2`, never `126` or `127`. Proves P1 to P6. Extra gate:
the architecture test allows `os/exec` in the plugin module only.

- Given a plugin whose bytes differ from its pin, when the source is used, then no process starts and the error is
  `retriever.connector.plugin_pin_mismatch`.
- Given a plugin that prints its environment, when it runs under the `integration` tag, then only its own credential
  appears.
- Given a plugin that never answers, when the limit passes, then the host kills it and exits `124`.

## 12. Profiles

The config paths gain the profile folder. The app adds `--profile` and `RETRIEVER_PROFILE`, and `sources list` and
`list_sources` name the active profile. The catalog format page gains the prefix rule. Proves F1 to F6.

- Given work and personal profiles, when a personal run starts, then the recording file system shows no read of the work
  folder.
- Given a personal catalog that names a work-prefixed variable, when it loads, then it fails.
- Given profiles and no choice, when any source command runs, then it exits `2` with `retriever.usage.profile_required`.
- Given `--profile ../other`, when the command runs, then it exits `2`.

## Follow-up outside these pull requests

Recording `docs/explanation/decisions/` in the repository adapter is a separate rule change through Rules
Propagation. It is not one of the twelve pull requests.
