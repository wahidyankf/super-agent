---
description: >-
  Indexes the practice rules: resource-aware development for local compute, and the task-tracking rule for multi-step
  work.
when_to_use: >-
  Use to find the rule for guarded local compute, remote dispatch, or keeping a task list current.
---

# Practice

## Directory Map

- [Resource-Aware Development](./resource-aware-development.md): every local build, test, and check runs through one
  checksum-pinned admission guard, with bounded recovery.
- [Resource-Aware Development Details](./resource-aware-development/README.md): the modules for Nx cache eviction,
  unadmitted tool runs, resource tiers, and remote dispatch.
- [Task Tracking](./task-tracking.md): a granular task list for every multi-step task, kept current as items resolve.
