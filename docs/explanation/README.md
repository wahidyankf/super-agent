---
title: Explanation
description: Why super-agent is shaped the way it is, with the scope, stability, data policy, and planned direction.
category: explanation
tags:
  - index
  - explanation
  - concepts
---

# Explanation

Why super-agent is built the way it is. These pages are for understanding. Nothing here is needed to use the
repository.

## Directory Map

- [What super-agent is](./what-super-agent-is.md) describes the human plus harness as the super agent, and what the
  project is not.
- [The sourcing layer](./sourcing-layer.md) explains why the `retriever` command-line tool reads sources through one
  catalog and fixed machine contracts.
- [Who guards what](./who-guards-what.md) explains which checks retriever makes itself and which the API token of a
  source makes.
- [Stability and versioning](./stability-and-versioning.md) explains what alpha means and how releases and breaking
  changes work.
- [Public data policy](./public-data-policy.md) explains why the repository is public and which kinds of data never
  enter it.
- [Software-factory note](./software-factory-note.md) records a later direction for orchestrating agents. It is a
  design note, and it is out of scope now.
- [Library layering](./library-layering.md) explains which Go modules may import which, and why the app and the
  libraries are split this way.

## Next steps

- [How-to guides](../how-to/) when you have a task to do.
- [Reference](../reference/) for exact names, gate ids, and the layout.
