# AGENTS.md

Instructions for coding agents and contributors. Use clear, simple English.

## Public Repository

This repository is public. Never commit:

- secrets, tokens, keys, or real `.env*` values (only `.env.example`, with empty values);
- real catalogs, `sources.yml` files, or user connector specs outside `examples/`;
- private host names, tailnet or LAN addresses, personal paths, or personal e-mail addresses;
- customer, employer, or tenant names.

Use neutral examples such as `example.internal` and `EXAMPLE_OUTLINE_API_KEY`. Commit metadata must use a GitHub
noreply address. Never set or change git identity or config.

## Delivery

`main` is the only integration branch. The bootstrap commit is the only direct commit to `main`; all later work goes
through a pull request from a worktree. Create worktrees only at `worktrees/<task>`, never in sibling directories.

## Commits

Use Conventional Commits with a lowercase type, an imperative subject, and no trailing period. Example:
`chore: bootstrap repository`.

## Working Rules

- Check `git status` before committing; stage only the files you intend to commit.
- Keep real values in uncommitted `.env.local` files.
- Keep generated output, caches, and local state out of Git (see `.gitignore`).
