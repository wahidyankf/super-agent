---
description: >-
  Keeps agent tools away from real environment files and secret directories, except the committed template, and states
  the staged-file guard that refuses to commit them.
when_to_use: >-
  Use when configuring an agent's tool permissions, a hook that inspects paths, a search index, or a guard that inspects
  staged files.
---

# Agent Environment-File Access

Agent tools must not read, write, or edit real environment files. That means every `.env*` file except `.env.example`.
Paths under `secrets/` and `credentials/` follow the same rule.

## Why

An agent's output can reach a transcript, a log, a pull request, or a model's context. A value that an agent read must
be treated as exposed, even if the agent never repeated it.

## Rules

1. Tool permissions deny read, write, and edit on protected paths. The denial lives in the harness configuration or a
   hook, not in the agent's good intentions.
2. An agent that needs a setting names the variable it needs and asks the owner to set the value in the real file. It
   does not open the file to check.
3. Search indexes and file listings exclude the same protected paths.
4. A staged-file guard refuses to commit a file that matches a real-environment pattern. An ignore rule alone is not a
   guard, because a forced add passes it. Review checks staged files as well.
5. A compute-admission exemption does not waive any rule on this page.

## Enforcement

The `env-validate` gate applies the staged-file rule that `repo-config.yml` declares. Agent-level denials are added when
agent definitions are adopted, and each harness binding must carry them. A harness without a way to deny a path is a gap
to report, not a reason to relax the rule.

The rule for secrets in general is in [No Secrets in Tracked Files](no-secrets-in-tracked-files.md). The variable naming
rules are in the [Environment Variable Contract](environment-variable-contract.md).
