---
title: Dispatch a task to another machine
description: How to run compute on a machine you control, following the remote-dispatch rule and its refusal path.
category: how-to
tags:
  - how-to
  - remote-dispatch
  - resources
---

# Dispatch a task to another machine

Use this guide to run compute on another machine that you control. The rule is in the [remote dispatch
module](../../repo-governance/development/practice/resource-aware-development/remote-dispatch.md), which sits inside
[resource-aware development](../../repo-governance/development/practice/resource-aware-development.md). This guide
lists the steps in order. The rule decides whether a dispatch is allowed.

## Keep the binding out of the repository

The binding names your machines and how to reach them. It lives outside this repository, at the path named by the
`REMOTE_DISPATCH_HOME` variable. The repository names the variable only. Never commit a host name, an address, an
alias, or the binding itself.

## Steps

1. **Check the binding.** `REMOTE_DISPATCH_HOME` must be set and readable. If it is unset or unreadable, do not
   dispatch. Record the refusal, and start no session, worktree, artifact, or compute on any target.
2. **Check the local reading.** Run `./hippo status` on the local machine. Dispatch only when the most recent local
   reading is non-normal, and record that reading.
3. **Check the target.** Immediately before any remote work, query the target. Its reading must be normal.
4. **Write the dispatch record.** Before remote work starts, write one record that names the dispatch and lists what
   the dispatch will create.
5. **Run exactly one HIPPO boundary, on the target.** Do not wrap the SSH command in a local HIPPO boundary. Do not
   nest a second boundary inside the first.
6. **Tear down.** Remove every artifact and worktree the task created on the target. If something remains, report
   what it is and why.
7. **Delete the record last.** Delete the dispatch record only after teardown is verified and nothing it lists
   remains.

Count each dispatch-created worktree toward the per-repository and global worktree caps. A dispatch waits, or moves
to another eligible machine, when the cap is full.

Never clean up state on a target by age alone. Remove state only after you confirm, on the target, that no live
session or process uses it.

## If something goes wrong

A refusal is a correct outcome. The `remote-dispatch-refusal` gate checks the refusal path, so a missing or
unreadable binding must stop the dispatch. See [Quality gates](../reference/quality-gates.md).
