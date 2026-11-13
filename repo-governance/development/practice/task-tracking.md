---
description: >-
  Requires a granular task list at the start of every multi-step task, updated as each item resolves, so that progress
  stays visible and no item is dropped silently.
when_to_use: >-
  Use at the start of any task with more than one step, and each time an item changes state.
---

# Task Tracking

A task with several steps is tracked as a list. The list is written before the work starts, and it is kept current as
each item resolves. A reader can see at any moment what is done, what is running, and what is blocked.

## Rules

1. **Write the list first.** Split the task into items small enough that each has one check that proves it done. Write
   them before the first change.
2. **One item in progress.** Mark an item in progress only when it is the one being worked on. Finish or block it before
   starting the next.
3. **Resolve each item.** An item ends as done, as blocked with the reason and the owner, or as dropped with the reason.
   It is never left in an unknown state.
4. **Add items as they appear.** A new step that the work reveals is added to the list at once, not kept in memory.
5. **Keep the list true.** When the plan changes, the list changes with it in the same step.

## Blocked Items

A blocked item names the decision or the fact that it waits for, and the person or role that can supply it. The task
moves on to an item it can finish, when one exists. A task that is blocked on every item stops and reports.

## Hand-off

The hand-off report shows the final state of the list: each item with its result, and each blocked or dropped item with
its reason. A list that was not kept current fails the hand-off, because the report would describe work that no longer
matches the record.

## Scope

The list lives in the task's own record: the plan, the pull request description, or the session. It is not a substitute
for the checks, and it is never marked done before its check passes.
