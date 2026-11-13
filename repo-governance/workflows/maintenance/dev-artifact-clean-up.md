---
description: >-
  Defines which development artifacts a delivered change leaves behind, which of them to remove at once, which to keep
  as evidence, and how to remove them without touching anything else.
when_to_use: >-
  Use after a task or plan finishes, after a pull request merges, or when the local checkout holds leftover worktrees,
  branches, or scratch files.
---

# Dev Artifact Clean-Up

A finished task leaves artifacts: worktrees, task branches, scratch files, caches, and evidence. Some are useful only
while the work runs. Others are the record of what was checked. This workflow removes the first kind and keeps the
second.

## Classes

| Class    | Examples                                                              | Action                         |
| -------- | --------------------------------------------------------------------- | ------------------------------ |
| scratch  | files under `local-tmp/`, notes, copies made for a single run         | remove after the task          |
| task     | the worktree, the task branch, the build output of that worktree      | remove after the merge         |
| evidence | check results, red and green records, the hand-off report             | keep where the plan says       |
| shared   | the primary checkout, the shared tool caches, other people's branches | never remove as part of a task |

When a class is unclear, treat the item as evidence, and keep it.

## Steps

1. **List the task's artifacts.** Name each worktree, branch, and folder that the task created. Use the plan or the pull
   request as the list, not memory.
2. **Confirm the merge.** Remove task artifacts only after their pull request has merged, or after the owner says the
   task is abandoned.
3. **Remove the scratch class.** Delete the files the task created under `local-tmp/`.
4. **Remove the task class.** Remove the worktree with `git worktree remove`, delete the task branch, and prune stale
   references.
5. **Keep the evidence class.** Move evidence to the location the plan declared. Never delete it to tidy up.
6. **Verify.** List the worktrees and branches again. Only the primary checkout and the branches that others own should
   remain.

## Removal Rules

- Remove only the names that the task created. Never remove by pattern alone, such as "every branch with a prefix".
- Never run a bulk prune that reaches beyond the task. Commands such as `git clean -fdx` on the primary checkout, or a
  global cache purge, can destroy another person's work or a shared cache.
- Never remove a worktree that a running session is bound to. Check first that no process uses it.
- Local caches are regenerable, so a stale cache may be removed. Keep anything unique to the work.

## Evidence in Ignored Locations

Evidence that must not be committed stays in the ignored `local-tmp/` folder, and the hand-off names its path relative
to the repository root. The [public repository data policy](../../conventions/security/public-repository-data-policy.md)
applies to anything that is later published.

## Enforcement

This workflow is applied by review. No gate removes artifacts, because a gate that deletes work would be a hazard.
