---
description: >-
  States that a result must come out the same from the same inputs on any checkout, so that every check, build, and
  release can be repeated and verified by someone else.
when_to_use: >-
  Use when designing a build, a check, a release, a fixture, or any step whose output another person must be able to
  reproduce.
---

# Reproducibility

A result that depends on who ran it, where, and when cannot be trusted. Reproducibility means the same inputs give the
same output on any clean checkout. A reviewer, a later maintainer, or an automated gate can then verify a claim without
taking the author's word.

## What It Requires

- **Pinned inputs.** Dependencies, tools, and releases are pinned by version and checksum, and the lock files are
  committed.
- **Declared inputs.** A build or check lists every file and setting it reads, so a cache cannot serve a stale result.
- **No hidden state.** A check does not depend on a local path, a machine name, a personal setting, or the time of day,
  unless that input is declared.
- **Recorded results.** Each check records what it ran and what it reported, so the result can be compared later.

## Tests and Fixtures

Tests create their own files and remove them. They use synthetic data that is the same on every machine. A test that
passes only on the author's machine is a defect in the test.

## Releases

A release is built from a tagged commit, with pinned tools, and its binaries carry checksums. Anyone can rebuild the
same binary from that commit and compare checksums. A release that cannot be rebuilt is not verified.

## Related

- [Root Cause Orientation](root-cause-orientation.md) requires a reproduction before a fix.
- [Quality Gate Contract](../development/workflow/quality-gate-contract.md) states how gates declare their inputs.
