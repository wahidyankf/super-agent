---
description: >-
  Defines the categories of data this public repository must never hold, in files, names, history, and release assets,
  and the screens every change passes before publication.
when_to_use: >-
  Use before adding a fixture, example, log, document, name, or release asset, and when deciding whether a value may be
  published.
---

# Public Repository Data Policy

This repository is public. Anything committed here is copied, cached, and indexed beyond the owner's control. The policy
therefore applies to file content, file and branch names, commit messages, history, and release assets alike.

## Forbidden Categories

Never publish any of the following. The list names categories only, so the policy holds without repeating any value.

- **Secrets and credentials.** API keys, tokens, passwords, session values, private keys, and connection strings that
  carry credentials.
- **Names of real organisations and people.** Employers, customers, tenants, groups, and private projects or
  repositories.
- **Network and host details.** Real host names, internal domain names, private network or local network addresses, and
  device names.
- **Personal paths and accounts.** Absolute paths under a home directory, usernames, and account identifiers.
- **Work and personal data.** Real catalogs, source lists, issue keys, records, messages, and documents from any real
  system.
- **Contact details.** Personal e-mail addresses and phone numbers. Commit author and committer identity is outside this
  list; see [git identity](../../development/workflow/git-identity.md).

## Synthetic Data Only

Fixtures, tests, documentation, and logs use values that nobody could mistake for real. Use the reserved example domain
names from [RFC 2606](https://www.rfc-editor.org/rfc/rfc2606), an `example.test` host name, placeholder names that
begin with `EXAMPLE_`, and invented people and organisations. A value copied from a real system is not synthetic,
however it is disguised.

## Screens

Three screens run before anything leaves the machine:

- the leak scan over the staged tree, before every commit and push;
- the leak scan over the full diff, on every pull request; and
- the leak scan over the whole history and every release asset, at release time.

The scan reports the file, the rule, and the location. It never reports the matched value. A scan that cannot run blocks
publication, because a screen that did not run passed nothing.

The only literals the scan accepts are the detection patterns in `scripts/public-safety/`. Those patterns find leaks;
they are not leaks. Nothing else is excepted.

## Review

A screen matches shapes, so a person reads each change before commit as well. The rule is in [Public Outbound
Safety](public-outbound-safety.md), and the procedure for a leak is in [No Secrets in Tracked
Files](no-secrets-in-tracked-files.md).

## If Something Leaks

Stop pushing. Tell the owner. Treat the value as exposed: rotate it first, then remove it from every reachable ref.
Record the commit, the path, and the rule that caught it, never the value.
