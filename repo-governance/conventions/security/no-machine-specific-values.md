---
description: >-
  Keeps absolute local paths, usernames, local addresses, and machine-dependent literals out of every tracked file,
  while allowing synthetic test data.
when_to_use: >-
  Use when committing scripts, configuration, fixtures, captured output, or documentation that could carry one machine's
  details.
---

# No Machine-Specific Values

A tracked file works on every checkout and describes no particular machine. This holds in every repository, public or
private: [Public Outbound Safety](public-outbound-safety.md) governs what is published, and this governs what is
committed at all.

## Two Harms

A machine-specific value breaks portability. The next checkout references a path that does not exist, and a script that
passed for its author fails for everyone else.

It also discloses. A username, a directory layout, or a network address stays in history for everyone who ever clones
the repository, long after the line that carried it was corrected.

## What Counts

- an absolute path under a home directory or a tool's install prefix: a repository-relative path, or one derived from
  the environment at runtime.
- a username, inside a path or as a literal identity in configuration: an environment variable or a documented
  placeholder.
- a local network address or a machine's hostname: configuration supplied at runtime.
- a connection string, key, token, or tool path that differs between machines and automation: an environment variable
  declared in the committed template.

Loopback addresses such as `127.0.0.1` and `localhost` may appear only in test configuration that targets a locally
running service, never beside a literal credential.

A script derives a location rather than recording the one its author had:

```bash
export TOOL_HOME="${TOOL_HOME:-$HOME/.tool}"
```

## Records Name Locations Portably

Plans, delivery records, and evidence identify a location relative to the repository root, as in
`docs/<topic>/<record>.md`, never by the resolved absolute path. Whoever runs the work resolves the route against their
own checkout. Evidence that genuinely needs a resolved absolute path is kept in an ignored location and never committed.

Captured tool output is the usual miss. Build, test, and coverage tools print resolved absolute paths, so a transcript
pasted verbatim carries the machine's layout although nobody typed it. Normalize the repository-root prefix to a
placeholder such as `<repository-path>` before committing it.

## Synthetic Test Data Is Allowed

A test value that exercises a format — an operating-system and architecture pair, a hostname fed to a parser — is test
data even when it resembles machine detail. A value copied from the author's machine because it was convenient is not,
however the test uses it.

For this rule, the question is where the value came from, not what it looks like. Where the repository is published, a
synthetic value must also be unmistakably synthetic, as [Public Outbound Safety](public-outbound-safety.md) requires.

## Machine-Dependent Values Belong in Configuration

Where a program needs a value that differs between machines, it reads an environment variable declared in the template
that [Environment Variable Contract](environment-variable-contract.md) defines. The real file stays ignored.

## Before Committing

Review the staged change for every category in What Counts, including every loopback address outside the
test-configuration allowance. Source, tests, fixtures, configuration, scripts, workflow definitions, generated files,
and documentation are all in scope; ignored files are not, because they are never committed. This review is required; a
pre-commit scan over staged additions is an enforcement point an adopter may add.

## If One Lands

1. Replace the value in the working tree with a relative path, a variable reference, or a placeholder.
2. Commit the correction.
3. If the value grants access, treat it as exposed and rotate it; a later commit does not remove it from history. [No
   Secrets in Tracked Files](no-secrets-in-tracked-files.md) owns that procedure.

A path or username that grants nothing needs only the corrective commit — unless it was already published, where [Public
Outbound Safety](public-outbound-safety.md) applies.
