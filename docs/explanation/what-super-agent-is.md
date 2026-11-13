---
title: What super-agent is
description: The human plus their harnesses is the super agent, served by one catalog and one access model.
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
and channels. Today it is a skeleton. The design below describes what it is for.

## It is not a catch-all agent

super-agent is not one agent that does everything. It does not decide for you, and it does not replace the agents
you already use. Each agent keeps its own job. super-agent supplies two shared things:

- **One catalog of sources.** Every agent finds the same connectors, so no agent keeps its own copy of a credential
  or a connection.
- **One access model.** Every agent meets the same access limits. Each connector checks access before it reads or
  writes data.

## How the pieces fit (planned)

```text
  you + harnesses ---> CLI or MCP server ---> connectors ---> data sources and channels
  (many agents)        (planned)              (planned)       (catalogs, documents, messages)
```

- **The harness side** is the command-line tool, `retriever-cli`, and a Model Context Protocol (MCP) server. Many
  agents use the same connectors through either one. Both are planned.
- **The data side** is the connectors. A connector reads and writes one kind of source. Both sides are planned.

Work and personal contexts stay separate in the design. Each has its own catalog and access ceiling. The
[software-factory note](./software-factory-note.md) lists this and the other planned items.

## What exists today

Only the repository skeleton exists: the quality gates, the governance rules, this documentation, and an empty plans
folder. There is no CLI, no MCP server, and no connector. The first command-line tool, `retriever-cli`, arrives in
the next pull request.

## Where to go next

- [Stability and versioning](./stability-and-versioning.md) explains what alpha means before you depend on the
  project.
- [Public data policy](./public-data-policy.md) explains what never enters the repository.
- [Software-factory note](./software-factory-note.md) records the later direction for orchestrating agents.
