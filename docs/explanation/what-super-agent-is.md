---
title: What super-agent is
description: The human plus their harnesses is the super agent, served by one catalog and one credential model.
category: explanation
tags:
  - explanation
  - concepts
---

# What super-agent is

## The human is the super agent

You work across many resources and channels, at work and outside work. You also use many coding agents to help you.
"Super agent" names you together with those agents and tools, which the project calls harnesses. It does not name a
single program.

super-agent amplifies that person. It is an orchestrator of many agents and of information from many data sources
and channels. Today it is an early alpha: a sourcing command-line tool and the repository around it. The design below
describes what it is for.

## It is not a catch-all agent

super-agent is not one agent that does everything. It does not decide for you, and it does not replace the agents
you already use. Each agent keeps its own job. super-agent supplies two shared things:

- **One catalog of sources.** Every agent finds the same connectors, so no agent keeps its own copy of a credential
  or a connection.
- **One credential model.** Every agent reaches a source with the same key handling and the same connection checks. The
  source's API token decides what a change may do. [Who guards what](./who-guards-what.md) explains the split.

## How the pieces fit

```text
  you + harnesses ---> CLI or MCP server ---> connectors ---> data sources and channels
  (many agents)        (built)                (built)         (wikis, test management, tickets)
```

- **The harness side** is the command-line tool, `retriever`, and its Model Context Protocol (MCP) server, started with
  `retriever mcp`. Many agents use the same connectors through either one.
- **The data side** is the connectors. The tool has three, for Outline wikis, Kiwi test management, and Jira Cloud. A
  connector reads one kind of source, and it changes a source only when the source's API token allows the change.
  Connectors for other kinds of channel are planned.

Work and personal contexts stay separate in the design. Each has its own catalog and its own API tokens. The
[software-factory note](./software-factory-note.md) lists this and the other planned items.

## What exists today

The repository holds the quality gates, the governance rules, this documentation, and an empty plans folder. It also
holds the first command-line tool, `retriever`, in `apps/retriever-cli`. The tool includes an MCP server and a
read-only GraphQL service, and it reads three kinds of source. Its interfaces are alpha, so read the
[changelog](../../CHANGELOG.md) before you depend on them.

## Where to go next

- [Stability and versioning](./stability-and-versioning.md) explains what alpha means before you depend on the
  project.
- [Public data policy](./public-data-policy.md) explains what never enters the repository.
- [The sourcing layer](./sourcing-layer.md) explains the command-line tool's design.
- [Software-factory note](./software-factory-note.md) records the later direction for orchestrating agents.
