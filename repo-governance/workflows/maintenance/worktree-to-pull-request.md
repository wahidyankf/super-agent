---
description: >-
  Defines how a change moves from a task worktree to a squash-merged pull request on main, the checks each step must
  pass, when an agent may merge, and the cleanup that follows.
when_to_use: >-
  Use when starting a task that changes the repository, opening or merging a pull request, or cleaning up after a merge.
---

# Worktree to Pull Request

`main` is the only integration branch. Every change after the bootstrap commit reaches `main` through a pull request
from a task worktree. No one commits directly to `main`.

## Worktrees

- Create a worktree only at `worktrees/<task>`, inside the repository root. Never create one in a sibling directory.
- Use one worktree for one task, on a branch named for that task.
- Give a worktree its own working directory and never change the directory of a session that is bound to it.

## Steps

1. **Start.** Create the worktree from an up-to-date `main`. Read the task's plan or request before changing anything.
2. **Change.** Build the change test-first, as [Test-Driven
   Development](../../development/workflow/test-driven-development.md) requires. Keep the commits thematic: one topic
   per commit.
3. **Check before commit.** Run `git status`, and stage only the files that belong to the change. Run the quick gate,
   and the leak scan over the staged tree.
4. **Commit.** Use a Conventional Commit: a lowercase type, an imperative subject, and no trailing period. Use the
   configured git identity, as [Git Identity](../../development/workflow/git-identity.md) requires.
5. **Push.** The push runs the leak scan over the pushed range. A refused push is fixed, never forced through.
6. **Open the pull request.** Give it a title and a body that state the change, the checks run, and the open questions.
   The leak scan runs over the title and body as well.
7. **Check the pull request.** Run the full pull-request gate locally through HIPPO and RHINO. Then poll CI.
8. **Merge.** Merge when the rules in the next section hold.
9. **Clean up.** Delete the worktree and the task branch, as the last section describes.

## Polling CI

Poll the status of the checks every 2 minutes, with `gh pr checks <number>`. Never use `gh run watch`, because it blocks
the session and hides the state that a later decision needs. Stop polling when every check has finished.

## When an Agent May Merge

An agent may squash-merge its own pull request into `main` only when all three of these hold:

1. every CI check is green;
2. the leak scan is clean over the tree and over the diff; and
3. the RHINO gate passes for the pull-request surface.

If any one fails, the agent does not merge. It reports the failure and the state of each check. An agent reports each
merge it makes, with the pull request number, the commit, and the checks that passed.

Until branch protection requires a pull request and green checks on `main`, these rules are the only guard. Protection
is enabled as a separate, reported step after the first pull request merges and CI is proven to run.

## Cleanup After Merge

After a merge, remove the work the task created:

- remove the worktree with `git worktree remove worktrees/<task>`;
- delete the local task branch, and the remote task branch when the merge deleted it not already;
- prune the stale remote references; and
- remove scratch files that the task left under `local-tmp/`, as [Dev Artifact Clean-Up](dev-artifact-clean-up.md)
  describes.

When the whole delivery is finished, every worktree and development artifact used to deliver it is removed.

## Refusals

- Never force-push to `main`, and never rewrite published history to hide a leak. A leak follows [No Secrets in Tracked
  Files](../../conventions/security/no-secrets-in-tracked-files.md).
- Never merge a pull request that another person or agent has marked as not ready, until that mark is resolved.
- Never use a command that changes the git identity or configuration to get a commit through.
