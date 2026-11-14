---
title: Reference
description: Exact facts about the repository and the retriever tool: names, gates, scripts, and the glossary.
category: reference
tags:
  - index
  - reference
---

# Reference

Exact statements about the repository as it is today. These pages describe facts rather than teach them. For a
walkthrough, start with the [how-to guides](../how-to/).

## Directory Map

- [Repository layout](./repository-layout.md) shows the real directory tree and what each top-level entry holds.
- [Environment variables](./environment-variables.md) lists variable names and says where values belong. It lists no
  values.
- [Quality gates](./quality-gates.md) lists every gate id in `repo-config.yml`, with the surfaces it runs on.
- [Scripts overview](./scripts-overview.md) summarises the wrapper scripts and points to their full annotations.
- [retriever commands](./retriever-cli-commands.md) lists the commands, global options, and flags of the `retriever`
  command-line tool.
- [retriever exit codes](./retriever-cli-exit-codes.md) lists every exit status and the commands that return it.
- [retriever JSON schemas](./retriever-cli-json-schemas.md) lists the JSON documents and the failure body.
- [retriever configuration](./retriever-cli-configuration.md) gives the precedence of the catalog and env-file paths.
- [Sources catalog format](./retriever-cli-catalog-format.md) gives the fields and rules of the sources file.
- [Migrating from Butler to retriever](./retriever-cli-migration-from-butler.md) maps each Butler behaviour to its
  retriever counterpart.
- [retriever scenario gap](./retriever-cli-scenario-gap.md) maps each Butler feature file to its retriever scenarios,
  and lists the gaps still to close.
- [retriever test gates](./retriever-cli-test-gates.md) lists the test targets, what the coverage gate measures and
  excludes, and what the binding check reports.
- [Go modules and tags](./go-modules-and-tags.md) lists the seven Go modules, their Nx projects and tags, their
  versions, and the tag format.
- [Operation and spec JSON Schema](./operation-json-schema.md) is PLANNED. It says what the schema will cover and that
  no schema file exists yet.

## Glossary

Terms used across the repository are defined in the [repository glossary](../../repo-governance/glossary.md).

## Next steps

- [How-to guides](../how-to/) to put these facts to work.
- [Explanation](../explanation/) for the reasons behind them.
