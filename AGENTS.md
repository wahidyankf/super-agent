# AGENTS.md

Instructions for coding agents and contributors. Use clear, simple English.

## Status

super-agent is alpha. Breaking changes are allowed, but each one must be listed in [CHANGELOG.md](./CHANGELOG.md).

## Public Repository

This repository is public. Never commit:

- secrets, tokens, keys, or real `.env*` values (only `.env.example`, with empty values);
- real catalogs, `sources.yml` files, or user connector specs outside `examples/`;
- private host names, tailnet or LAN addresses, personal paths, or personal e-mail addresses;
- customer, employer, or tenant names.

Use neutral examples such as `example.test` and `EXAMPLE_OUTLINE_API_KEY`. See the
[public outbound safety](./repo-governance/conventions/security/public-outbound-safety.md) and
[no machine-specific values](./repo-governance/conventions/security/no-machine-specific-values.md) conventions.

Use the configured git identity for commits; never set or change git identity or config.

## Delivery

`main` is the only integration branch. The bootstrap commit is the only direct commit to `main`; all later work goes
through a pull request from a worktree. Create worktrees only at `worktrees/<task>`, never in sibling directories.

Run the quality gates through HIPPO and RHINO:

```bash
./hippo run --class ephemeral --resource-tier standard --disk-path . -- ./rhino gate run --surface pull-request
```

## Remote Dispatch

Dispatch compute to another machine only through the binding named by `REMOTE_DISPATCH_HOME`. The rule and its
lifecycle are in [remote dispatch](./repo-governance/development/practice/resource-aware-development/remote-dispatch.md).
Exactly one HIPPO boundary runs on the target host, as
[resource-aware development](./repo-governance/development/practice/resource-aware-development.md) describes.

## Commits

Use Conventional Commits with a lowercase type, an imperative subject, and no trailing period. Example:
`chore: bootstrap repository`.

## Working Rules

- Check `git status` before committing; stage only the files you intend to commit.
- Keep real values in uncommitted `.env.local` files.
- Keep generated output, caches, and local state out of Git (see `.gitignore`).
- Hand-author canonical sources; never hand-edit generated harness adapters.
