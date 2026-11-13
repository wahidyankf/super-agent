---
description: >-
  Requires agents to commit with the git identity already configured on the machine, and never to set or change git
  identity or configuration.
when_to_use: >-
  Use before any commit, when a tool asks for an author name or e-mail, or when a commit is refused for identity
  reasons.
---

# Git Identity

Commits use the git identity that the owner has configured on the machine. Agents never change that configuration.

## Rules

1. Use the configured git identity. Do not pass `-c user.name`, `-c user.email`, or `--author` to override it.
2. Never set or change git identity or configuration. Do not write `user.*` values, edit a git configuration file, or
   set an environment variable that overrides the identity.
3. When a commit fails because no identity is configured, stop and report it. Do not invent an identity.
4. The author and committer identity of a commit is not a leak, and the screens for published data do not check it. The
   [public outbound safety](../../conventions/security/public-outbound-safety.md) convention lists what is screened.
