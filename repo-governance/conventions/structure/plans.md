---
description: >-
  Defines the plans/ folder: when a plan may be written, how plans move through their lifecycle folders, and what a plan
  must hold.
when_to_use: >-
  Use before writing a plan, moving a plan between lifecycle folders, or closing one out.
---

# Plans

A plan describes work to do: its goal, its steps, and how each step is checked. Plans live in `plans/`.

## When a Plan May Be Written

A plan is written only when the owner asks for one. Plan mode in a harness does not count as that request. Without the
request, the work is described in the reply or in the pull request.

## Lifecycle Folders

- `plans/backlog/` holds an accepted idea that has not started.
- `plans/in-progress/` holds a plan whose work has started.
- `plans/done/` holds a plan whose every step has been checked.

A plan moves to `in-progress` when its work starts and to `done` when its last check passes. A plan that is abandoned
records why in its own text before it moves to `done`. The folders are created by the first plan that needs them.

## Required Content

Each plan states these five parts, in this order:

1. **Goal:** one paragraph that names the outcome.
2. **Scope:** what is in and what is out.
3. **Steps:** an ordered list, where each step names its check.
4. **Decisions:** open questions, with the owner's answer when one exists.
5. **Evidence:** where the results of each check are recorded.

## Rules

- A plan links to the specs and conventions it depends on. It does not copy their text.
- A plan that stops matching the code is updated or closed. It is never left to drift.
- A plan names no private value and uses repository-relative paths, as the [public repository data
  policy](../security/public-repository-data-policy.md) requires.
- A plan carries no time estimate.
