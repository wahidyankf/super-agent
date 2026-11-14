---
title: The sourcing layer
description: Why retriever reads sources through one catalog, one key file, and fixed machine contracts.
category: explanation
tags:
  - explanation
  - cli
  - retriever
  - design
---

# The sourcing layer

`retriever` is the part of super-agent that reads content from other services. It reads Outline wikis, Kiwi test
management, and Jira Cloud. It can also change a source, and the source's API token decides whether the source
accepts the change. This page explains the design. The exact contracts are in the reference pages.

## The problem it solves

A person who works across several services needs the same content in several places. A script needs it, a coding agent
needs it, and a dashboard needs it. Each caller that holds its own copy of a service address and a key multiplies the
places where a key can leak, and it multiplies the code that must change when a service moves. The sourcing layer keeps
one copy of that knowledge.

## One catalog, one key file

The sources catalog names each source, its kind, its tags, and the variable that holds its key. It never holds
a key. The key stays in the process environment or in a local env file that Git ignores. A caller asks for a source by
its ID, and it never sees the key. The `sources list` output prints neither the endpoint nor the key's name.
A missing-key error names the variable, but it never prints a value. See
[configuration](../reference/retriever-cli-configuration.md) and
[the catalog format](../reference/retriever-cli-catalog-format.md).

The catalog holds no access level. Tags only label a source, and a catalog that still has an `access` key is refused
when it loads. What a source accepts is decided by its API token. [Who guards what](who-guards-what.md) explains the
split.

## Three machine contracts

The same operations reach a caller through three interfaces. Each one is a contract, not a convenience.

- **The command line** prints text for a person, or one JSON document for a script with `--output json`. A failure
  prints one JSON body on standard error. Each exit status has a fixed meaning, and the set of statuses is closed. See
  [exit codes](../reference/retriever-cli-exit-codes.md) and [JSON schemas](../reference/retriever-cli-json-schemas.md).
- **The MCP server**, started with `retriever mcp`, offers the sources as tools to a Model Context Protocol client. Its
  tools are the tools of the catalog, and `--tools` limits them to the named resources.
- **The GraphQL service**, started with `retriever serve`, answers read queries on the loopback address only. It has no
  mutation, introspection is off, and a query that is too deep or too large is refused before any source is called.

A dashboard that reads content is the intended use of the GraphQL service. Nothing in this repository builds that
dashboard yet.

Every JSON document carries a `schemaVersion`. A field is added without a version change, and a removed or renamed
field raises the version. That rule lets a caller depend on the contract with confidence.

## What it leaves out

The tool does not take on the work of the older Butler tool that it replaces. These parts stay in the tool that owns
them:

- the vault layout and its front matter, so an export is plain Markdown with no front matter;
- PDF rendering, job reports, and GitHub commands;
- any built-in source. Every source comes from a catalog.

It reads only the environment variables listed in
[environment variables](../reference/environment-variables.md).

## Alpha status

The tool is alpha. A release may break a command, a flag, or a field. Each break is listed in the
[changelog](../../CHANGELOG.md), and the [migration table](../reference/retriever-cli-migration-from-butler.md) maps the
old Butler behaviour to the new one.

## Related

- [Add a source from the example catalog](../how-to/add-a-source-from-the-example-catalog.md) is the first-run guide.
- [Commands](../reference/retriever-cli-commands.md) lists the commands and flags.
