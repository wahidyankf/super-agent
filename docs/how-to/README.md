---
title: How-to guides
description: Guides for using the retriever tool, delivering changes, running gates, repinning tools, and releases.
category: how-to
tags:
  - index
  - how-to
---

# How-to guides

Directions for a goal you already have. Each guide assumes you know what you want and gets you there.

If you are still learning what super-agent is, start with the [explanation pages](../explanation/).

## Directory Map

- [Add a source from the example catalog](./add-a-source-from-the-example-catalog.md) builds `retriever`, points it at
  a catalog, gives it a key, and reads a document.
- [Deliver a change with a pull request](./deliver-a-change-with-a-pull-request.md) creates a worktree, opens a pull
  request, waits for CI, and merges under the merge rules.
- [Run the quality gates](./run-the-quality-gates.md) runs the gates through HIPPO and RHINO, locally and in Git
  hooks.
- [Repin HIPPO and RHINO](./repin-hippo-and-rhino.md) moves the lock-file pins to a new release and verifies them.
- [Dispatch a task to another machine](./dispatch-a-task-to-another-machine.md) runs compute on a machine you
  control, under the remote-dispatch rule.
- [Cut an alpha pre-release](./cut-an-alpha-pre-release.md) prepares the changelog, the tag, the checksum list, and
  the GitHub pre-release.
- [Add a connector](./add-a-connector.md) adds a source connector as a library module, wires it into the app, and tests
  it.
- [Add a spec connector](./add-a-spec-connector.md) adds a source kind with one YAML spec file, without a Go change.

## Next steps

- [Reference](../reference/) for exact names, gate ids, and the repository layout.
- [Explanation](../explanation/) for why the project works this way.
