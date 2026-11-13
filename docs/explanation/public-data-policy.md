---
title: Public data policy
description: Why the repository is public, which data never enters it, and what to do if something leaks.
category: explanation
tags:
  - explanation
  - security
  - policy
---

# Public data policy

The super-agent repository is public on GitHub. Everything it publishes is visible to anyone: file contents, file
names, commit messages, branch and tag names, release notes, and release assets. This page explains what stays out.

## Why the repository is public

The code is MIT-licensed and published so that others can read it, use it, and check its releases. A public
repository has one cost. Every byte becomes public, so the rules below apply to everything the repository holds,
including history and release assets.

## What never enters the repository

The rules name categories only. None of them lists a real value.

- **Secrets.** Tokens, API keys, passwords, private keys, and the values of any `.env*` file. Only `.env.example` is
  tracked, and its values stay empty.
- **Real catalogs and connector specs.** Your own catalog of sources, any `sources.yml`, and any connector spec
  written for your accounts. Examples live only under `examples/`, and they use synthetic data.
- **Machine-specific values.** Host names, network addresses (private ranges included), personal file-system paths,
  and the names of your own machines.
- **Organisation and person data.** Customer, employer, and tenant names, and any personal data about a real person.
- **Fixtures, tests, documentation, and logs.** These use synthetic data only.

## Commit metadata

Commit author and committer metadata use the configured git identity. This is an owner decision. The identity can
include an e-mail address, and that address becomes public with each commit. No other category on this page is
relaxed. Agents never set or change git identity or git config.

## Use neutral examples

Write examples with placeholders, such as `<private-host>` for a host name, `EXAMPLE_OUTLINE_API_KEY` for a
variable name, and `<api-token>` for a value. A placeholder must not hide what the example means. If the example
cannot work without real data, it does not belong in a public repository.

## How the rules are checked

- The `public-safety-tree` gate screens the tracked tree and its file names.
- The `public-safety-range` gate screens each commit in a pushed or pull-request range, one commit at a time.
- The `public-safety-commit-message` gate screens commit messages.
- The scans look for shapes such as home-directory paths, private address ranges, and internal host-name suffixes.
  They also look for credential-shaped values.
- Release notes and assets are screened before they are published.
- A scan that cannot run is a failure. It never counts as a pass.

The gates are listed in [Quality gates](../reference/quality-gates.md). The scripts are described in [Scripts
overview](../reference/scripts-overview.md).

## If something leaks

Stop. Do not push anything more. Tell the repository's owner at once. A value that reached the public history needs
a history rewrite to remove it, and copies that were already cloned or scraped keep the value.
