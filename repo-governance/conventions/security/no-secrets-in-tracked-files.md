---
description: >-
  Forbids any secret in any tracked file or durable change text, whatever the repository's visibility or remote, and
  makes rotation the only remediation for a leak.
when_to_use: >-
  Use when deciding where a credential belongs, before committing configuration, or when a secret may already be in
  history.
---

# No Secrets in Tracked Files

No secret enters a tracked file. There is no exception for a private repository, a repository with no remote, a test, a
temporary commit, or urgency.

## What Is a Secret

Anything that lets its holder into a system or service: private keys, certificates, passwords, API tokens, session
material, recovery codes, privileged account names, and connection strings.

## Why Visibility Does Not Matter

Version history is permanent and copied. A committed value lives in every clone, backup, sync copy, mirror, and editor
cache that ever held the repository, and none of those copies can be cleaned from the repository.

Private visibility narrows who can read today. It does not narrow who reads after a visibility change, a fork, a leaked
backup, or a lost laptop, and a repository with no remote is still copied by backups and sync services. A rule scoped to
public repositories protects only what [Public Outbound Safety](public-outbound-safety.md) already covers and leaves
every other copy open.

## Where Real Values Live

Outside version history, in the locations recorded by the census that [Environment Variable
Contract](environment-variable-contract.md) defines: real environment files, an ignored secrets directory, a platform's
secret store, or a password manager. The committed template holds names and placeholders only, and agent access to real
files is governed by [Agent Environment-File Access](agent-env-file-access.md).

An ignore rule reduces accidental tracking; it is not a security boundary, because a forced add walks straight past it.
The staged-file guard that [Agent Environment-File Access](agent-env-file-access.md) defines refuses that add only for a
file matching its recorded real-environment pattern. A file forced in from any other ignored location, such as a secrets
directory, is stopped by review: every repository reviews each staged change for secrets before it commits. A secret
scan over staged changes is a further enforcement point an adopter may add for such files.

## Durable Text Counts

Text written around a change stays readable long after the change merges: an issue body, a commit message, a chat
thread, a branch name, a review comment, a pull-request description. No secret belongs in any of them.

To ask for access or for help, state what access is needed and what it is for, and leave every value out. Share
specifics only once the person receiving them is confirmed to be authorized for that access. A value posted to explain a
problem is exposed the moment it is posted.

## Rotation Is the Only Remediation

A secret that reached history is compromised, whether or not anyone is known to have seen it. Rotation is the remedy;
removing the value afterwards is cleanup, and rewriting history never substitutes for rotation.

1. **Rotate first.** Revoke or rotate the value at its provider before anything else. Every minute spent on a rewrite
   before rotation is a minute the value is still live.
2. **Inventory the reach.** Every branch, tag, release, pull request, and mirror that contains the commit, and every
   other surface holding the same value.
3. **Remove it from every reachable ref.** Remove the value from the working tree, then from every ref that still
   reaches it, not one branch alone. Whoever holds authority over that history performs the rewrite.
4. **Hold merges until it is gone.** While any ref or pull request still carrying the value is reachable, no ordinary
   merge lands. Such a pull request is replaced by a clean one, and the host is asked to purge cached copies where it
   can.
5. **Record without the value.** Incident notes carry commit identifiers, paths, and provider references — never the
   value, a fragment of it, or a command line containing it.

State plainly what cannot be undone. External clones, forks, and caches are beyond reach, and rotation is what made them
harmless.

## Agent Tool Access

Agent tools must not read, write, or edit real environment files or secret directories. The scope and the enforcement
points are in [Agent Environment-File Access](agent-env-file-access.md). Compute-admission exemptions do not waive these
restrictions.
