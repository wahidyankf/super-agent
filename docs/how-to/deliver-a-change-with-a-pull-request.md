---
title: Deliver a change with a pull request
description: How to deliver one change through a worktree, a pull request, CI polling, a squash merge, and cleanup.
category: how-to
tags:
  - how-to
  - delivery
  - git
---

# Deliver a change with a pull request

Every change after the bootstrap commit goes through a pull request from a worktree. The bootstrap commit is the
only direct commit to `main`. One task has one branch, one worktree, and one pull request.

## 1. Create the worktree

Create the worktree only at `worktrees/<task>` inside this repository. Never create one in a sibling directory.

```sh
git worktree add worktrees/<task> -b <branch> main
```

The `worktrees/` folder is ignored by Git, so a checkout there is never committed. Do the rest of the work inside
the worktree.

## 2. Make the change

Keep the change focused on one purpose. Run the gates before you commit. See [Run the quality
gates](./run-the-quality-gates.md).

## 3. Commit

- Stage only the files you mean to commit. Check with `git status` first.
- Use Conventional Commits: a lowercase type, an imperative subject, and no trailing period.
  Example: `docs: add explanation pages`.
- Use the configured git identity. Never set or change git identity or git config.
- The commit-msg and pre-commit hooks run the gates. The pre-commit hook can reformat staged files. Review those
  changes and commit again if it changed anything.

## 4. Push and open the pull request

```sh
git push -u origin <branch>
gh pr create --base main --head <branch> --title "<conventional title>" --body "<what changed and why>"
```

## 5. Wait for CI, polling every two minutes

Check the pull request every two minutes:

```sh
gh pr checks <number>
```

Do not use `gh run watch`.

## 6. Merge only when every condition holds

Squash-merge into `main` only when all three conditions hold:

1. Every CI check is green.
2. The leak scan is clean over the tree and over the full diff. The `public-safety-tree` and `public-safety-range`
   gates pass.
3. The RHINO gate passes for the `pull-request` surface.

```sh
gh pr merge <number> --squash
```

Report every merge to the repository owner.

Branch protection on `main` is planned for after the first pull request merges. Until it is enabled, these
conditions are followed by convention.

## 7. Clean up after the merge

```sh
git worktree remove worktrees/<task>
git branch -D <branch>
```

A squash merge does not mark the branch as merged, so `git branch -d` refuses to delete it. Confirm that the pull
request merged, then use `-D`. Also remove any scratch files the task left under `local-tmp/`.
