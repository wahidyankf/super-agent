---
title: Stability and versioning
description: What alpha means for super-agent, how its version numbers work, and how to consume a release safely.
category: explanation
tags:
  - explanation
  - versioning
  - release
---

# Stability and versioning

super-agent is alpha. Read this page before you depend on it.

## What alpha means

- Breaking changes are allowed in any release, including minor and patch releases, until a stable 1.0.
- Commands, flags, file formats, connector specs, JSON output, and APIs can change in any release.
- Do not depend on super-agent in production.

## Version numbers

Releases use the form `v0.x.y`. Under semantic versioning, a `0.x` release makes no promise of compatibility. Any
release can break an earlier one. The project stays on `0.x` until its maintainer decides otherwise.

## Where breaking changes are listed

Each breaking change is listed in [CHANGELOG.md](../../CHANGELOG.md) under the release that introduces it, with a
**Breaking** label. Before you upgrade, read every entry between your pinned version and the new one.

## Releases

Releases are tags on GitHub. Each one is a pre-release, and GitHub does not mark a pre-release as the latest
release. Each release publishes a checksum list, so you can verify every archive. The steps are in [Cut an alpha
pre-release](../how-to/cut-an-alpha-pre-release.md). No release has been published yet.

## How to consume a release

1. Pin an exact version. Never follow `main` at runtime.
2. Pin the SHA-256 checksum of each archive you use, and verify it before you run the file.
3. Read the changelog entries between your pinned version and the new one.
4. Change the pin in one reviewed change, so the upgrade is easy to find and to undo.

The same rules apply to the tools this repository pins. See [Repin HIPPO and
RHINO](../how-to/repin-hippo-and-rhino.md).
