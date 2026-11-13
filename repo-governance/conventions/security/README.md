---
description: >-
  Indexes the security conventions that keep secrets, machine details, private data, and private keys out of tracked and
  published files.
when_to_use: >-
  Use to find the rule for a secret, a machine-specific value, a published artifact, an environment file, or a private
  key.
---

# Security Conventions

These conventions apply to every tracked file and every published artifact. Read the public data policy before
publishing anything, and read the others when their topic applies.

## Directory Map

- [Public Repository Data Policy](./public-repository-data-policy.md): the categories of data this public repository
  must never hold, and the screens that run before publication.
- [Public Outbound Safety](./public-outbound-safety.md): which outbound artifacts a screen must read, and why no bypass
  exists.
- [No Secrets in Tracked Files](./no-secrets-in-tracked-files.md): why no secret enters a tracked file, and why rotation
  is the only remedy for a leak.
- [No Machine-Specific Values](./no-machine-specific-values.md): why tracked files hold no absolute path, username, or
  local address.
- [Environment Variable Contract](./environment-variable-contract.md): where variables are declared, how they are named,
  and where real values live.
- [Agent Environment-File Access](./agent-env-file-access.md): which environment and secret paths agent tools must not
  touch.
- [Private Key Access](./private-key-access.md): the rule that agents never open or print a private key.
